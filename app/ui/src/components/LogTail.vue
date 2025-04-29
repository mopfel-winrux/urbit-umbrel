<template>
  <section v-if="state.urbitRunning" style="white-space:pre-wrap">
    <h2>Ship logs</h2>
    <p>uptime: {{ formatUptime(log.uptime) }}</p>
    <div class="logs-container">
      <pre ref="logsElement" v-text="log.tail" class="logs" @scroll="handleScroll"></pre>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import { getLogs } from '../api'

const props = defineProps({
  state: {
    type: Object,
    default: () => ({})
  }
})

const log = ref({ running: false, tail: '' })
const logsElement = ref(null)
const userHasScrolled = ref(false)

function formatUptime(seconds) {
  if (!seconds && seconds !== 0) return 'unknown'
  
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  
  const parts = []
  if (days > 0) parts.push(`${days}d`)
  if (hours > 0) parts.push(`${hours}h`)
  if (minutes > 0) parts.push(`${minutes}m`)
  if (remainingSeconds > 0 || parts.length === 0) parts.push(`${remainingSeconds}s`)
  
  return parts.join(' ')
}

function scrollToBottom() {
  if (!logsElement.value) return
  
  nextTick(() => {
    if (logsElement.value && !userHasScrolled.value) {
      logsElement.value.scrollTop = logsElement.value.scrollHeight
    }
  })
}

function handleScroll() {
  if (!logsElement.value) return
  
  const { scrollTop, scrollHeight, clientHeight } = logsElement.value
  const isAtBottom = scrollHeight - scrollTop - clientHeight < 10
  
  userHasScrolled.value = !isAtBottom
}

async function poll() {
  try {
    const newLog = await getLogs() || {}
    const isAtBottom = logsElement.value ? 
      (logsElement.value.scrollHeight - logsElement.value.scrollTop - logsElement.value.clientHeight < 10) : 
      false
    log.value = newLog
    if (isAtBottom) {
      userHasScrolled.value = false
      scrollToBottom()
    }
    
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  }
}

onMounted(() => {
  poll()
  const intervalId = setInterval(poll, 5000)
  scrollToBottom()
  return () => {
    clearInterval(intervalId)
  }
})

watch(() => log.value.tail, (newValue, oldValue) => {
  if (newValue !== oldValue) {
    scrollToBottom()
  }
}, { flush: 'post' })
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