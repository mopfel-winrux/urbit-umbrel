<template>
  <main v-if="ready" class="app">
    <status :state="state" />

    <div v-if="!state.urbitRunning" class="grid">
      <info />
      <boot-existing
        :state="state"
        :migration="migration"
        :disabled="state.urbitRunning"
        @boot="bootSelected"
        @migrate="migrateSelected"
      />
      <boot-comet :disabled="state.urbitRunning" @boot="bootCometSelected" />
      <upload-key :disabled="state.urbitRunning" @done="refresh" />
      <upload-pier :disabled="state.urbitRunning" @done="refresh" />
    </div>

    <div v-else>
      <log-tail :state="state" />
    </div>
  </main>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { boot, bootComet, getMigrationOptions, getStatus, migrateVere } from '../api'

import Status from '../components/Status.vue'
import UploadKey from '../components/UploadKey.vue'
import UploadPier from '../components/UploadPier.vue'
import BootExisting from '../components/BootExisting.vue'
import BootComet from '../components/BootComet.vue'
import LogTail from '../components/LogTail.vue'
import Info from '../components/Info.vue'

const state = ref({})
const migration = ref({
  loading: true,
  versions: [],
  currentTag: '',
  currentVersion: '',
  error: '',
})
const ready = ref(false)

let poller

async function refresh() {
  try {
    state.value = await getStatus() || {}
    if (Object.keys(state.value).length) ready.value = true
  } catch { }
}

async function refreshMigration() {
  migration.value = { ...migration.value, loading: true }
  try {
    const data = await getMigrationOptions()
    migration.value = {
      loading: false,
      versions: data?.versions || [],
      currentTag: data?.currentTag || '',
      currentVersion: data?.currentVersion || '',
      error: data?.error || '',
    }
  } catch {
    migration.value = {
      ...migration.value,
      loading: false,
      error: 'Could not load vere releases from GitHub.',
    }
  }
}

async function bootSelected(path, loom) {
  await boot(path, loom)
  await refresh()
}

async function migrateSelected(path, loom, version) {
  await migrateVere(path, loom, version)
  await refresh()
}

async function bootCometSelected(loom) {
  await bootComet(loom)
  await refresh()
}

onMounted(async () => {
  await Promise.all([refresh(), refreshMigration()])
  poller = setInterval(refresh, 1_000)
})

onUnmounted(() => {
  clearInterval(poller)
})
</script>
