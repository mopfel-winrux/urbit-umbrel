<template>
  <section>
    <h2>Boot new comet</h2>
    <p>Don't have an ID? <a href="https://docs.urbit.org/glossary/comet" target="_blank">Comets↗</a> are disposable free identities.</p> 
    <div class="choices">
        <span class="label">Loom:</span>
        <div class="options-container">
            <label v-for="l in [31,32,33]" :key="l" class="option">
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
        <button @click="go" :disabled="disabled">⊙ boot comet</button>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { bootComet } from '../api'

const props = defineProps({
  state:    { type: Object, required: true },
  disabled: Boolean,
})
const emits = defineEmits(['boot'])
const loom = ref(31)
async function go () {
  await bootComet(loom.value)
  emits('boot')
}
</script>
<style scoped>
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
</style>