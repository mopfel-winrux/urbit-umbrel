<template>
  <main class="login">
    <form @submit.prevent="go">
      <input
        v-model="pass"
        type="password"
        placeholder="App password"
        autocomplete="current-password"
        autofocus
        @input="err = false"
      />

      <button :disabled="busy">
        <span v-if="busy" class="spinner"></span>
        Login
      </button>

      <p v-if="err" class="bad">Bad password</p>
    </form>
  </main>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login, getStatus } from '../api'

const pass = ref('')
const err = ref(false)
const busy = ref(false)
const router = useRouter()

async function go() {
  if (!pass.value.trim()) return
  busy.value = true
  
  try {
    const ok = await login(pass.value.trim())
    
    if (ok) {
      console.log("Login successful, getting status...");
      localStorage.setItem('authenticated', 'true'); 
      
      try {
        await getStatus(); 
        console.log("Status check successful, navigating to home");
        router.replace('/');
      } catch (error) {
        console.error("Status check failed:", error);
        err.value = true;
      }
    } else {
      err.value = true;
      pass.value = '';
    }
  } catch (error) {
    console.error("Login error:", error);
    err.value = true;
  } finally {
    busy.value = false;
  }
}
</script>

<style scoped>
.login {
  max-width: 22rem;
  margin: 6rem auto;
  text-align: center;
}
.login form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.login input {
  padding: .6rem;
  font-family: inherit;
  font-size: 1rem;
}
.spinner {
  display: inline-block;
  width: .8rem;
  height: .8rem;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  vertical-align: -1px;
  margin-right: .4rem;
}
.bad { color: #c00; font-size: .9rem }

@keyframes spin { to{ transform:rotate(360deg) } }
</style>
