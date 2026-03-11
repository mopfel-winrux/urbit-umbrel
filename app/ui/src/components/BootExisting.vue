<template>
  <section class="boot-card">
    <div class="card-head">
      <div>
        <h2>Boot pier</h2>
        <p class="lede">Start a keyfile or an existing pier with the bundled runtime.</p>
      </div>
    </div>

    <p v-if="opts.length === 0" class="empty">
      No keyfiles or piers yet. Upload one first.
    </p>

    <form v-else @submit.prevent="go">
      <div class="choice-row">
        <span class="label">Piers</span>
        <div class="options-container">
          <label v-for="p in opts" :key="p" class="option">
            <input
              type="radio"
              :value="p"
              v-model="path"
              :disabled="disabled"
            />
            <span>{{ base(p) }}</span>
          </label>
        </div>
      </div>

      <div class="choice-row">
        <span class="label">Loom</span>
        <div class="options-container">
          <label v-for="l in state.loomValues" :key="l" class="option">
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
        <button :disabled="disabled || !path">⊙ boot</button>

        <div class="migrate-cluster">
          <button
            type="button"
            class="migrate-toggle"
            :disabled="disabled || migration.loading || !selectedIsPier"
            @click="toggleMigration"
          >
            Migrate vere
          </button>

          <span
            class="help-tip"
            tabindex="0"
            aria-label="Use this if your ship displays runtime or event log version errors."
          >
            ?
            <span class="tooltip">
              Use this if your ship displays runtime or event log version errors.
            </span>
          </span>
        </div>
      </div>
    </form>

    <transition name="panel-fade">
      <div v-if="showMigration" class="migration-panel">
        <p class="eyebrow">Replay Event Log</p>
        <h3>Migrate through older <code>vere</code> releases before boot</h3>
        <p class="migration-copy">
          Start from a stable runtime as far back as <code>3.5</code>, replay forward through each later release, then hand off to the <code>vere</code> runtime bundled with this app.
        </p>

        <p v-if="migration.loading" class="migration-note">
          Loading release list from GitHub…
        </p>
        <p v-else-if="migration.error" class="migration-error">
          {{ migration.error }}
        </p>
        <p v-else-if="!selectedIsPier" class="migration-note">
          Select an existing pier first. Migration replay does not apply to keyfiles.
        </p>
        <template v-else>
          <label class="version-picker">
            <span>Starting version</span>
            <select v-model="migrationTag" :disabled="disabled || migration.loading">
              <option v-for="version in migrationVersions" :key="version.tag" :value="version.tag">
                {{ version.version }}{{ version.tag === migration.currentTag ? ' (current)' : '' }}
              </option>
            </select>
          </label>

          <p class="migration-meta">
            Bundled runtime: <strong>{{ migration.currentVersion || 'unknown' }}</strong>
          </p>

          <div class="migration-actions">
            <button
              type="button"
              class="replay-button"
              :disabled="migrationActionDisabled"
              @click="runMigration"
            >
              Start replay
            </button>
            <button type="button" class="ghost-button" @click="showMigration = false">
              Cancel
            </button>
          </div>
        </template>
      </div>
    </transition>
  </section>
</template>

<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  state: { type: Object, required: true },
  migration: { type: Object, default: () => ({ versions: [] }) },
  disabled: Boolean,
})
const emit = defineEmits(['boot', 'migrate'])

const path = ref('')
const loom = ref(props.state.loomValues?.[0] || 31)
const migrationTag = ref('')
const showMigration = ref(false)

const opts = computed(() => [
  ...(props.state.keys ?? []),
  ...(props.state.piers ?? []),
])

const migrationVersions = computed(() => props.migration?.versions ?? [])
const selectedIsPier = computed(() => (props.state.piers ?? []).includes(path.value))
const migrationActionDisabled = computed(() =>
  props.disabled ||
  props.migration?.loading ||
  !selectedIsPier.value ||
  !migrationTag.value
)

watch(opts, list => {
  if (!list.includes(path.value)) path.value = ''
})

watch(
  () => props.state.loomValues,
  list => {
    if (Array.isArray(list) && list.length && !list.includes(loom.value)) {
      loom.value = list[0]
    }
  },
  { immediate: true }
)

watch(selectedIsPier, isPier => {
  if (!isPier) showMigration.value = false
})

watch(
  [migrationVersions, () => props.migration?.currentTag],
  () => {
    if (!migrationVersions.value.some(version => version.tag === migrationTag.value)) {
      migrationTag.value = defaultMigrationTag()
    }
  },
  { immediate: true }
)

const base = p => {
  const trimmed = p.endsWith('/') ? p.slice(0, -1) : p
  return trimmed.split('/').pop()
}

function defaultMigrationTag() {
  const versions = migrationVersions.value
  if (!versions.length) return ''

  const currentIndex = versions.findIndex(version => version.tag === props.migration?.currentTag)
  if (currentIndex === 0 && versions[1]) return versions[1].tag
  return versions[0].tag
}

function go() {
  if (!path.value) return
  emit('boot', path.value, loom.value)
}

function toggleMigration() {
  if (!selectedIsPier.value) return
  showMigration.value = !showMigration.value
  if (showMigration.value && !migrationTag.value) {
    migrationTag.value = defaultMigrationTag()
  }
}

function runMigration() {
  if (migrationActionDisabled.value) return
  emit('migrate', path.value, loom.value, migrationTag.value)
}
</script>

<style scoped>
.boot-card {
  position: relative;
  overflow: hidden;
}

.card-head h2,
.migration-panel h3 {
  margin: 0;
}

.lede,
.empty,
.migration-copy,
.migration-note,
.migration-error,
.migration-meta {
  margin: 0;
}

code {
  padding: 0.14rem 0.35rem;
  border-radius: 8px;
  background: rgba(13, 59, 102, 0.08);
}

.card-head {
  margin-bottom: 1.25rem;
}

.lede {
  margin-top: 0.35rem;
  color: var(--accent-lite);
  line-height: 1.5;
}

.empty {
  color: var(--accent-lite);
}

.choice-row {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.label {
  min-width: 68px;
  padding-top: 0.55rem;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--accent-lite);
}

.options-container {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: 0.65rem;
}

.option {
  display: inline-flex;
  align-items: center;
}

.option input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.option span {
  padding: 0.35rem 0.6rem;
  border-radius: 4px;
  border: 1px solid var(--chip-border);
  background: var(--chip-bg);
  transition: transform 0.15s ease, border-color 0.15s ease, background 0.15s ease;
}

.option input:checked + span {
  border-color: var(--accent);
  background: #f3f4f6;
  transform: translateY(-1px);
}

.actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-top: 1.35rem;
  flex-wrap: wrap;
}

.migrate-cluster {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.migrate-toggle {
  background: var(--accent);
}

.help-tip {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  border: 1px solid var(--card-border);
  background: #fff;
  color: var(--accent);
  font-weight: 700;
  cursor: help;
}

.tooltip {
  position: absolute;
  right: 0;
  bottom: calc(100% + 0.85rem);
  width: 15rem;
  padding: 0.75rem 0.9rem;
  border-radius: 6px;
  background: #2c3e50;
  color: #fff;
  font-size: 0.78rem;
  line-height: 1.45;
  box-shadow: 0 8px 16px rgba(27,31,35,0.15);
  opacity: 0;
  pointer-events: none;
  transform: translateY(6px);
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.help-tip:hover .tooltip,
.help-tip:focus .tooltip,
.help-tip:focus-within .tooltip {
  opacity: 1;
  transform: translateY(0);
}

.migration-panel {
  margin-top: 1.35rem;
  padding: 1.1rem;
  border-radius: 8px;
  border: 1px solid var(--card-border);
  background: #fafafa;
  box-shadow: none;
}

.eyebrow {
  margin: 0 0 0.55rem;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--accent-lite);
}

.migration-copy {
  margin-top: 0.55rem;
  line-height: 1.55;
  color: var(--accent-lite);
}

.version-picker {
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  max-width: 16rem;
  margin-top: 1rem;
}

.version-picker span {
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent-lite);
}

.migration-meta,
.migration-note,
.migration-error {
  margin-top: 0.95rem;
  line-height: 1.5;
}

.migration-error {
  color: #9f1239;
}

.migration-actions {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-top: 1rem;
}

.replay-button {
  background: var(--accent);
}

.ghost-button {
  background: #fff;
  color: var(--accent);
  border: 1px solid var(--card-border);
}

.panel-fade-enter-active,
.panel-fade-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.panel-fade-enter-from,
.panel-fade-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (max-width: 640px) {
  .choice-row {
    flex-direction: column;
    gap: 0.6rem;
  }

  .label {
    min-width: 0;
    padding-top: 0;
  }

  .actions {
    align-items: stretch;
  }

  .migrate-cluster {
    justify-content: space-between;
    width: 100%;
  }

  .tooltip {
    width: 12.5rem;
  }
}
</style>
