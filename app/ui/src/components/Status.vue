<template>
  <section v-if="state.urbitRunning">
    <p>
      Ship is {{ statusLabel }}<span v-if="state.code">: <code>{{ state.code }}</code></span>
      <span v-else-if="state.state === 'booting'" class="spinner">(~)</span>
    </p>

    <button v-if="state.urbitRunning" @click="stopAndRefresh">
      ⊙ stop
    </button>
    <button v-if="state.code" @click="launchPage" :disabled="launching">
      ⊙ launch
    </button>
  </section>
</template>

<script setup>
import { ref, computed } from 'vue'
import { getStatus, stopUrbit } from '../api'

const props = defineProps({ state: Object })
const launching = ref(false)
const statusLabel = computed(() => props.state.code ? 'running' : 'booting')

async function stopAndRefresh() {
  await stopUrbit()
  const s = await getStatus()
  Object.assign(props.state, s)
}

async function launchPage() {
  launching.value = true
  const form = new URLSearchParams()
  form.append('password', props.state.code.trim())
  form.append('redirect', '/')
  await fetch('/~/login', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: form.toString(),
  })
  window.location.href = '/'
}
</script>

<style scoped>
button { margin-right: .5rem; margin-top: .5rem }
.spinner {
  display: inline-block;
  margin-left: 0.5rem;
  font-family: monospace;
  animation: spin 1.5s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>
