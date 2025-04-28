<template>
  <section>
    <h2>Upload pier archive (zip/tar.gz/tgz)</h2>
    <p>example for a pier at ~/urbit/mister-dozzod:
      <br><code>tar czf mister-dozzod.tar.gz -C ~/urbit mister-dozzod</code>
    </p>
    <input
      type="file"
      @change="onPick"
      accept=".zip,.tar.gz,.tgz"
      :disabled="disabled || busy"
    />
    <button
      @click="send"
      :disabled="disabled || busy || !file"
    >
      ⊙ upload
    </button>
    <p v-if="progress">{{ progress }}%</p>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { uploadPier } from '../api'

const props = defineProps({ disabled: Boolean })
const emit = defineEmits(['done'])

const file     = ref(null)
const progress = ref(0)
const busy     = ref(false)

function onPick(e) {
  file.value = e.target.files[0]
}

async function send() {
  if (!file.value) return
  busy.value = true
  const chunk = 5 * 1024 * 1024
  const total = Math.ceil(file.value.size / chunk)

  for (let i = 0; i < total; i++) {
    const fd = new FormData()
    fd.append('file', file.value.slice(i * chunk, (i + 1) * chunk), file.value.name)
    fd.append('dzchunkindex', i)
    fd.append('dztotalchunkcount', total)
    fd.append('dzchunkbyteoffset', i * chunk)
    fd.append('dztotalfilesize', file.value.size)

    await uploadPier(fd)
    progress.value = Math.round(((i + 1) / total) * 100)
  }

  file.value = null
  progress.value = 0
  busy.value = false
  emit('done')
}
</script>

<style scoped>
button {
  margin-top: 0.5rem;
  margin-right: 0.5rem;
}
</style>
