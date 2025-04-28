<template>
  <section>
    <h2>Upload keyfile</h2>
    <p>Upload a keyfile from <a href="https://bridge.urbit.org">Bridge↗</a> or a <a href="https://docs.urbit.org/manual/os/basics#moons">moon↗</a>.</p>
    <p><code>Don't have a planet? Buy one with BTC at <a href="https://subject.network/buy" target="_blank">subject.network↗</a></code></p>
    <input type="file" @change="onPick" accept=".key" :disabled="disabled || busy" />
    <button @click="send" :disabled="disabled || busy || !file">⊙ upload</button>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { uploadKey } from '../api'

const props = defineProps({ disabled: Boolean })
const emits = defineEmits(['done'])

const file = ref(null)
const busy = ref(false)

function onPick (e) {
  file.value = e.target.files[0]
}

async function send () {
  if (!file.value) return
  busy.value = true
  try {
    await uploadKey(file.value)
    emits('done')
    file.value = null
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.wrap {
  max-width: 48rem;
  margin: 1.5rem auto 0;
  padding: 0 10px;
}
.hero {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 1rem 10px;
  border: 2px solid var(--accent);
  border-radius: 6px;
  background: var(--card-bg);
}
.logo {
  width: 4rem;
  height: auto;
}
.title {
  flex: 1;
  text-align: center;
}
.title h1 {
  margin: 0;
  font-size: 1.6rem;
}
.title h4 {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 400;
  color: var(--accent-lite);
  display: flex;
  align-items: center;
  justify-content: center;
}
.inline-logo {
  height: 1em;
  width: auto;
}

@media (max-width: 480px) {
  .hero {
    flex-direction: column;
  }
}
</style>
