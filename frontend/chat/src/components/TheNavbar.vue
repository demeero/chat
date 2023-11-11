<template>
  <nav aria-label="main navigation" class="navbar is-dark" role="navigation">
    <div class="navbar-brand">
      <router-link class="navbar-item" to="/">
        <span class="icon-text">
          <span class="icon">
            <i class="fas fa-home"></i>
          </span>
          <span>Home</span>
        </span>
      </router-link>
      <a aria-expanded="false" aria-label="menu" class="navbar-burger" data-target="navbarBasicExample"
         role="button">
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
        <span aria-hidden="true"></span>
      </a>
    </div>

    <div class="navbar-menu is-active">
      <div class="navbar-start">
      </div>
      <div class="navbar-end">
        <!--        <router-link class="navbar-item" v-if="user.username"-->
        <!--                     :to="{ name: 'user-details', params: {id: user.username}, replace: true}">-->
        <!--          <span class="icon-text">-->
        <!--              <span class="icon">-->
        <!--                <i class="fas fa-regular fa-id-card"></i>-->
        <!--              </span>-->
        <!--              <span>{{ user.username }}</span>-->
        <!--          </span>-->
        <!--        </router-link>-->
        <div class="navbar-item">
          <div class="buttons">
            <a v-if="active" class="button" @click="logout">
              <span class="icon">
              <i class="fas fa-solid fa-right-from-bracket"></i>
            </span>
              <span>Log out</span>
            </a>
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>
<script>
import {useToast} from 'vue-toastification';
import {useUserStore} from '@/stores/user';

export default {
  name: 'TheNavbar',
  computed: {
    active() {
      return useUserStore().session?.active;
    }
  },
  methods: {
    async logout() {
      try {
        await useUserStore().logout()
      } catch (e) {
        console.error('failed logout', e);
      }
      useToast().success('Logged out successfully!');
      this.$router.push({name: 'signin'});
    }
  },
};
</script>
