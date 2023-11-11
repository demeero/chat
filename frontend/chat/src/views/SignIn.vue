<template>
  <div class="columns is-centered">
    <div class="column is-4">
      <form class="box has-background-white is-centered" @submit.prevent="signIn">
        <h2 class="title has-text-centered has-text-dark">
          Signin
        </h2>
        <div class="field">
          <label class="label">Email</label>
          <p class="control">
            <input v-model="email" class="input" placeholder="foo@bar.com" required type="email"/>
          </p>
        </div>
        <div class="field">
          <label class="label">Password</label>
          <p class="control">
            <input v-model="password" class="input" placeholder="******" required type="password"/>
          </p>
        </div>
        <div class="field">
          <p class="control has-text-centered">
            <button class="button is-primary" type="submit">Login</button>
          </p>
        </div>
        <div class="field">
          <p class="control has-text-centered">
            <button class="button is-info" type="button" @click="register">Register</button>
          </p>
        </div>
      </form>
    </div>
  </div>
</template>
<script>
import {useUserStore} from '@/stores/user';
import {useToast} from "vue-toastification";

export default {
  name: 'SignIn',
  data() {
    return {
      email: null,
      password: null,
    };
  },
  methods: {
    register() {
      this.$router.push('/signup');
    },
    async signIn() {
      const toast = useToast();
      const userStore = useUserStore();
      try {
        await userStore.login(this.email, this.password);
        toast.success('Successfully logged in!');
        this.$router.push('/');
      } catch (err) {
        console.error('failed login', err);
        toast.error(err.message);
      }
    },
  },
};
</script>
