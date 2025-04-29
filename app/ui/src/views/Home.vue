<template>
  <main v-if="ready" class="app">
    <status :state="state" @stop="stopUrbit" />

    <div v-if="!state.urbitRunning" class="grid">
      <info />
      <boot-existing :state="state" :disabled="state.urbitRunning" @boot="boot"/>
      <boot-comet :disabled="state.urbitRunning" @boot="bootComet"/>
      <upload-key   :disabled="state.urbitRunning" @done="refresh"/>
      <upload-pier  :disabled="state.urbitRunning" @done="refresh"/>
    </div>

    <div v-else>
      <log-tail :state="state" />
    </div>
  </main>
</template>


<script setup>
import { ref, onMounted } from 'vue'
import { getStatus, stopUrbit, resetCode, boot, bootComet } from '../api'

import Status from '../components/Status.vue'
import UploadKey from '../components/UploadKey.vue'
import UploadPier from '../components/UploadPier.vue'
import BootExisting from '../components/BootExisting.vue'
import BootComet from '../components/BootComet.vue'
import LogTail from '../components/LogTail.vue'
import Info from '../components/Info.vue'

const state = ref({})
const ready = ref(false)

async function refresh () {
  try {
    state.value = await getStatus() || {}
    if (Object.keys(state.value).length) ready.value = true
  } catch { }
}

onMounted(refresh)
setInterval(refresh, 1_000)
</script>
