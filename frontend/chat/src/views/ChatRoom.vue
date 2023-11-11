<script setup>
import ChatBox from "@/components/ChatBox.vue";
import ChatMessageSender from "@/components/ChatMessageSender.vue";
</script>

<template>
  <div class="container has-text-centered">
    <h3 class="title is-3">Chat Room</h3>
  </div>

  <div class="columns is-centered">
    <div class="column is-7 is-clipped">
      <ChatBox :loadingHistory="loadingHistory" :msgs="msgs" :userId="userId" @scrollArrivedTop="onArrivedTop"/>
      <ChatMessageSender :userId="userId"/>
    </div>
  </div>
</template>
<script>
import Chat from '@/service/chat'
import {useUserStore} from "@/stores/user";
import {useWebSocket} from '@vueuse/core'
import {useToast} from "vue-toastification";

export default {
  name: 'ChatRoom',
  data() {
    return {
      loadingHistory: false,
      msgs: [],
      receiverWS: null,
      userId: '',
      nextPageToken: '',
      wasLastPage: false,
    };
  },
  beforeUnmount() {
    this.receiverWS.close();
  },
  async created() {
    this.userId = useUserStore().session?.identity?.id;
    this.receiverWS = useWebSocket(import.meta.env.VITE_CHAT_RECEIVER_WS_URL, {
      autoReconnect: {
        retries: 5,
        delay: 1000,
        onFailed() {
          useToast().error('Failed to connect WebSocket after retries')
        },
      },
      onConnected: () => console.log('receiver ws connected'),
      onDisconnected: () => console.log('receiver ws disconnected'),
      onError: (err) => console.error('receiver ws error', err),
      onMessage: async (_, msg) => {
        console.log('receiver ws msg data', msg.data)
        this.msgs.push(JSON.parse(msg.data))
      },
    })
  },
  async mounted() {
    await this.loadHistory()
  },
  methods: {
    async loadHistory() {
      try {
        this.loadingHistory = true;
        const chatSvc = new Chat()
        const data = await chatSvc.loadHistory(this.nextPageToken, 20)
        this.msgs = data.page.concat(this.msgs)
        this.nextPageToken = data.next_page_token
        if (!this.nextPageToken) {
          this.wasLastPage = true
        }
      } catch (err) {
        if (err.status === 401) {
          await useUserStore().logout()
          return this.$router.push({name: 'signin'})
        }
        useToast().error(err.message);
        console.error(err)
      }
      this.loadingHistory = false;
    },
    async onArrivedTop() {
      if (!this.wasLastPage) {
        await this.loadHistory()
      }
    }
  },
};
</script>
