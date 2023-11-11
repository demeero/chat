<template>

  <div v-if="active" class="box has-background-white">
    <aside class="menu">
      <p class="menu-label has-text-grey-dark">
        Chats
      </p>
      <ul class="menu-list">
        <li>
          <a class="has-text-grey-dark">
            Personal
          </a>
        </li>
        <li>
          <a class="has-text-grey-dark">
            Channels
          </a>
        </li>
      </ul>
    </aside>
  </div>
</template>

<script>
import {useUserStore} from '@/stores/user';

export default {
  name: 'TheNavbar',
  computed: {
    active() {
      return useUserStore().session.active;
    }
  },
};
</script>
