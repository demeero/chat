<script setup>
import ChatMessage from "@/components/ChatMessage.vue";
import {vScroll} from '@vueuse/components'
</script>
<template>
  <div ref="chatBox" v-scroll="onScroll" class="box" style="overflow-y: scroll; max-height: 600px;">
    <progress v-if="loadingHistory" class="progress is-large is-info" max="100">60%</progress>
    <ChatMessage v-for="m in msgs" :key="m.pending_id" :msg=m :userId="userId"/>
    <span ref="chatBottom"></span>
  </div>
</template>
<script>
import {useScroll} from "@vueuse/core/index";

export default {
  name: 'ChatRoom',
  emits: ['scrollArrivedTop'],
  props: {
    userId: {
      type: String,
      required: true,
    },
    loadingHistory: {
      type: Boolean,
      required: true,
    },
    msgs: {
      type: Array,
      required: true,
    },
  },

  mounted() {
    this.chatBoxScroll = useScroll(this.$refs.chatBox, {offset: {top: 50, bottom: 100}})
    this.scrollDown()
  },

  watch: {
    msgs: {
      async handler(newMsgs, oldMsgs) {
        if (oldMsgs.length === 0 || this.chatBoxScroll?.arrivedState.bottom) {
          return this.scrollDown()
        }
      },
      deep: true,
      flush: 'post',
    },
  },
  methods: {
    async onScroll(evt) {
      if (this.chatBoxScroll?.arrivedState.top && !this.chatBoxScroll?.arrivedState.bottom) {
        this.$emit('scrollArrivedTop')
      }
    },
    scrollDown() {
      this.$refs.chatBottom.scrollIntoView({behavior: 'instant', block: 'end'})
    }
  },
}
</script>
