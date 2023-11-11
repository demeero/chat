<script setup>
import {RouterView} from 'vue-router'
import TheNavbar from "./components/TheNavbar.vue";
import TheFooter from "./components/TheFooter.vue";
</script>

<template>
  <TheNavbar/>
  <section class="section">
    <div class="container">
      <router-view>
      </router-view>
    </div>
  </section>
  <TheFooter/>
  <!--  <div class="columns is-centered">-->
  <!--    <div class="column is-one-fifth">-->
  <!--      <TheSidebar/>-->
  <!--    </div>-->
  <!--    <div class="column">-->
  <!--      <section class="section">-->
  <!--        <div class="container">-->
  <!--          <RouterView/>-->
  <!--        </div>-->
  <!--      </section>-->
  <!--    </div>-->
  <!--  </div>-->
</template>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

nav {
  padding: 30px;
}

nav a {
  font-weight: bold;
  color: #2c3e50;
}

nav a.router-link-exact-active {
  color: #42b983;
}
</style>
