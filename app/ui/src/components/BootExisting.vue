<template>
  <section>
    <h2>Boot pier</h2>

    <p v-if="opts.length === 0">
      No keyfiles or piers yet — upload one first.
    </p>

    <form v-else @submit.prevent="go">
      <div class="choices">
        <span class="label">Piers:</span>
        <div class="options-container">
          <label v-for="p in opts" :key="p" class="option">
            <input
              type="radio"
              :value="p"
              v-model="path"
              :disabled="disabled"
            />
            {{ base(p) }}
          </label>
        </div>
      </div>

      <div class="choices">
        <span class="label">Loom:</span>
        <div class="options-container">
          <label v-for="l in state.loomValues" :key="l" class="option">
            <input
              type="radio"
              :value="l"
              v-model.number="loom"
              :disabled="disabled"
            />
            {{ l }}
          </label>
        </div>
      </div>

      <div class="actions">
        <button :disabled="disabled || !path">⊙ boot</button>
      </div>
    </form>
  </section>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  state:    { type: Object, required: true },
  disabled: Boolean,
})
const emit = defineEmits(['boot'])

const path = ref('')
const loom = ref(props.state.loomValues?.[0] || 31)

const opts = computed(() => [
  ...(props.state.keys  ?? []),
  ...(props.state.piers ?? []),
])

watch(opts, list => {
  if (!list.includes(path.value)) path.value = ''
})

const base = p => {
  const trimmed = p.endsWith('/') ? p.slice(0, -1) : p
  return trimmed.split('/').pop()
}

function go() {
  if (!path.value) return
  emit('boot', path.value, loom.value)
}
</script>

<style scoped>
.choices { 
  display: flex; 
  margin-bottom: 1rem;
  align-items: flex-start;
}
.label { 
  font-weight: bold; 
  margin-right: 1rem;
  min-width: 60px;
}
.options-container {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  flex: 1;
}
.option {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  margin-bottom: 0.25rem;
}
.actions { 
  display: flex; 
  justify-content: flex-end; 
  margin-top: 1rem; 
}
</style>