<template>
  <p :class="msg.user.id === userId ? 'has-text-left' : 'has-text-right'" class="content"
     style="overflow-wrap: anywhere;">
    <span class="is-size-6">{{ msg.user.first_name }} {{ msg.user.last_name }}</span>
    <span class="is-size-7 has-text-weight-light ml-1">{{ msg.user.email }}</span>
    <br/>
    <span :class="msg.user.id === userId ? 'is-success' : 'is-info'" class="tag tag-msg is-medium">
            {{ msg.msg }}
          </span>
    <br/>
    <span
        class="is-size-7 is-italic has-text-weight-light">{{ new Date(msg.created_at).toLocaleString() }}</span>
  </p>
</template>
<script>
export default {
  name: 'ChatMessage',
  props: {
    userId: {
      type: String,
      required: true,
    },
    msg: {
      type: Object,
      required: true,
      properties: {
        pending_id: {
          type: String,
          required: true,
        },
        id: {
          type: Number,
        },
        msg: {
          type: String,
          required: true,
        },
        user: {
          type: Object,
          required: true,
          properties: {
            id: {
              type: Number,
              required: true,
            },
            email: {
              type: String,
              required: true,
            },
            first_name: {
              type: String,
            },
            last_name: {
              type: String,
            },
          },
        },
        created_at: {
          type: String,
        },
      },
    },
  }
}
</script>
<style>
.tag-msg {
  height: auto !important;
  white-space: pre-wrap !important;
  max-width: 50%;
}
</style>
