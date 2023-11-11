<template>
  <div class="container">
    <div class="container has-text-centered">
      <h3 class="title is-3">Create User</h3>
    </div>

    <div class="columns is-centered">
      <div class="column is-6-tablet is-5-desktop is-4-widescreen">
        <form class="box" @submit.prevent="createUser">
          <div class="field">
            <label class="label">Email</label>
            <div class="control">
              <input v-model="email" class="input" placeholder="foo@bar.com" required type="email">
            </div>
          </div>
          <div class="columns is-centered is-multiline">
            <div class="column">
              <div class="field">
                <label class="label">Password</label>
                <div class="control">
                  <input v-model="pass1" class="input" placeholder="******" type="password">
                </div>
              </div>
            </div>
            <div class="column">
              <div class="field">
                <label class="label">Confirm Password</label>
                <div class="control">
                  <input v-model="pass2" class="input" placeholder="*****" required type="password">
                </div>
              </div>
            </div>
          </div>
          <div class="field">
            <label class="label">First Name</label>
            <div class="control">
              <input v-model="firstName" class="input" type="text">
            </div>
          </div>
          <div class="field">
            <label class="label">Last Name</label>
            <div class="control">
              <input v-model="lastName" class="input" type="text">
            </div>
          </div>
          <button class="button is-primary" type="submit">Submit</button>
        </form>
      </div>
    </div>
  </div>
</template>
<script>
import {useToast} from 'vue-toastification';
import {useUserStore} from '@/stores/user';

export default {
  name: 'UserSignUp',
  data() {
    return {
      email: null,
      pass1: null,
      pass2: null,
      firstName: null,
      lastName: null,
    };
  },
  methods: {
    async createUser() {
      const toast = useToast();
      if (this.pass1 !== this.pass2) {
        toast.error('Passwords do not match');
        return;
      }
      try {
        await useUserStore().register({
          email: this.email,
          password: this.pass1,
          firstName: this.firstName,
          lastName: this.lastName,
        });
        toast.success('Successfully registered!');
        this.$router.push('/');
      } catch (err) {
        toast.error(err.message);
      }
    },
  },
};
</script>
