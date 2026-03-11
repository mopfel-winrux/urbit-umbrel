<template>
  <section>
    <h2>Boot new comet</h2>
    <p>Don't have an ID? <a href="https://docs.urbit.org/glossary/comet" target="_blank">Comets↗</a> are disposable free identities.</p>
    <div class="choices">
      <span class="label">Loom:</span>
      <div class="options-container">
        <label v-for="l in [31, 32, 33]" :key="l" class="option">
          <input
            type="radio"
            :value="l"
            v-model.number="loom"
            :disabled="disabled"
          />
          <span>{{ l }}</span>
        </label>
      </div>
    </div>
    <div class="actions">
      <button @click="go" :disabled="disabled">⊙ boot comet</button>
    </div>
  </section>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  disabled: Boolean,
})
const emits = defineEmits(['boot'])
const loom = ref(31)

function go() {
  emits('boot', loom.value)
}
</script>

<style scoped>
.option {
  display: inline-flex;
  align-items: center;
  margin-bottom: 0.25rem;
}

.option input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.option span {
  padding: 0.45rem 0.85rem;
  border-radius: 999px;
  border: 1px solid var(--chip-border);
  background: var(--chip-bg);
  transition: transform 0.15s ease, border-color 0.15s ease, background 0.15s ease;
}

.option input:checked + span {
  border-color: var(--accent);
  background: rgba(13, 59, 102, 0.12);
  transform: translateY(-1px);
}

.actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 1rem;
}

.choices {
  display: flex;
  margin-bottom: 1rem;
  align-items: flex-start;
}

.label {
  font-weight: bold;
  margin-right: 1rem;
  min-width: 60px;
  padding-top: 0.45rem;
}

.options-container {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
