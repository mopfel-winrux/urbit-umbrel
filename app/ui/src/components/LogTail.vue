<template>
  <section v-if="log.running" style="white-space:pre-wrap">
    <h2>Ship logs</h2>
    <p>uptime: {{ log.uptime }} s</p>
    <div class="logs-container">
      <pre ref="logsElement" v-text="log.tail" class="logs"></pre>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import { getLogs } from '../api'

const log = ref({ running: false })
const logsElement = ref(null)

async function poll() {
  log.value = await getLogs() || {}
}

function scrollToBottom() {
  if (logsElement.value) {
    nextTick(() => {
      logsElement.value.scrollTop = logsElement.value.scrollHeight
    })
  }
}

watch(() => log.value.tail, () => {
  scrollToBottom()
})

onMounted(() => {
  poll()
  setInterval(poll, 5000)
})
</script>
<style>
.logs-container {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #eee;
  border-radius: 4px;
  background: #f9f9f9;
}

.logs {
  word-wrap: break-word;
  margin: 0;
  padding: 10px;
  max-width: 90%;
}
</style>