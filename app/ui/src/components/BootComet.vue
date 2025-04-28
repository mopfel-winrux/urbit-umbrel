<template>
  <section>
    <h2>Boot new comet</h2>
    <p>Don't have an ID? <a href="https://docs.urbit.org/glossary/comet" target="_blank">Comets↗</a> are disposable free identities.</p> 
    <p>loom size</p>
    <label v-for="l in [31,32,33]" :key="l">
      <input type="radio" name="loomc" :value="l" v-model="loom" /> {{ l }}
    </label>
    <button @click="go" :disabled="disabled">⊙ boot comet</button>
  </section>
</template>

<script setup>
import { ref } from 'vue'
import { bootComet } from '../api'

const props = defineProps({ disabled:Boolean })
const emits = defineEmits(['boot'])

const loom = ref(31)
async function go () {
  await bootComet(loom.value)
  emits('boot')
}
</script>
