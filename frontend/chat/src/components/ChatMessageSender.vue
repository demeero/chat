<template>
  <div class="columns is-centered is-gapless">
    <div class="column is-four-fifths">
          <textarea v-model=msg class="textarea is-small" placeholder="Your message here..." required
                    rows="2"></textarea>
    </div>
    <div class="column">
      <button :disabled="senderWS?.status !== 'OPEN' || !msg" class="button is-primary is-responsive"
              type="submit" @click="send">Send
      </button>
    </div>
  </div>
</template>
<script>
import {useWebSocket} from "@vueuse/core";
import {useToast} from "vue-toastification";

export default {
  name: 'ChatMessageSender',
  props: {
    userId: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      msg: '',
      senderWS: null,
    }
  },
  created() {
    this.senderWS = useWebSocket(import.meta.env.VITE_CHAT_SENDER_WS_URL, {
      autoReconnect: {
        retries: 5,
        delay: 1000,
        onFailed() {
          useToast().error('Failed to connect sender WebSocket after retries')
        },
      },
      onConnected: () => console.log('sender ws connected'),
      onDisconnected: () => console.log('sender ws disconnected'),
      onError: (err) => console.error('sender ws error', err)
    })
  },
  beforeUnmount() {
    this.senderWS?.close();
  },
  methods: {
    async send() {
      this.senderWS?.send(JSON.stringify({pending_id: this.userId + Date.now(), msg: this.msg}))
      this.msg = ''
    },
  },
}
</script>
