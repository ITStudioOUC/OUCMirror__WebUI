<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ContentRecord } from '../api'
import { validGeneratorVersions, versionedGenerators, type GeneratorRelease, type GeneratorVersions, type VersionedGenerator } from '../generator-catalog'
import { mirrorGenerators } from '../mirror-config'
import { useGeneratorCatalog } from '../useGeneratorCatalog'
import MirrorConfig from './MirrorConfig.vue'

const props = defineProps<{ records: ContentRecord<GeneratorVersions>[]; busy: boolean }>()
const emit = defineEmits<{ save: [record: ContentRecord<GeneratorVersions>] }>()
const { t } = useI18n()
const { catalog } = useGeneratorCatalog()
const selected = ref<VersionedGenerator>('ubuntu')
const draft = ref<ContentRecord<GeneratorVersions> | null>(null)
watch([selected, () => props.records], () => {
  const record = props.records.find(record => record.id === selected.value)
  draft.value = record ? JSON.parse(JSON.stringify(record)) : null
}, { immediate: true })
const valid = computed(() => draft.value && validGeneratorVersions(selected.value, draft.value.data))
const previewCatalog = computed(() => valid.value && draft.value && catalog.value ? { ...catalog.value, [selected.value]: draft.value.data } : null)
function addRelease() {
  const release: GeneratorRelease = { id: '', label: '' }
  if (selected.value === 'debian') Object.assign(release, { stable: true, firmware: true, backports: false })
  if (selected.value === 'proxmox') release.keyring = '/usr/share/keyrings/proxmox-archive-keyring.gpg'
  draft.value?.data.releases.push(release)
}
function removeRelease(index: number) {
  if (!draft.value) return
  const [removed] = draft.value.data.releases.splice(index, 1)
  if (removed?.id === draft.value.data.defaultRelease) draft.value.data.defaultRelease = draft.value.data.releases[0]?.id || ''
}
function moveRelease(index: number, offset: number) {
  const rows = draft.value?.data.releases
  if (!rows || index + offset < 0 || index + offset >= rows.length) return
  const [release] = rows.splice(index, 1)
  if (release) rows.splice(index + offset, 0, release)
}
</script>

<template>
  <div class="generator-editor">
    <p>{{ t('admin.generatorIntro') }}</p>
    <label>{{ t('admin.configGenerator') }}<select v-model="selected" :disabled="busy"><option v-for="type in versionedGenerators" :key="type" :value="type">{{ t(mirrorGenerators[type].nameKey) }}</option></select></label>
    <form v-if="draft" @submit.prevent="valid && emit('save', draft)">
      <fieldset :disabled="busy">
        <label>{{ t('admin.defaultRelease') }}<select v-model="draft.data.defaultRelease" required><option v-for="(release, index) in draft.data.releases" :key="index" :value="release.id">{{ release.label || release.id || t('admin.unnamedRelease', { index: index + 1 }) }}</option></select></label>
        <p class="generator-hint">{{ t(`admin.generatorHint_${selected}`) }}</p>
        <article v-for="(release, index) in draft.data.releases" :key="index" class="generator-release">
          <div class="generator-fields">
            <label>{{ t('admin.releaseLabel') }}<input v-model="release.label" maxlength="160" required /></label>
            <label>{{ t('admin.releaseId') }}<input v-model="release.id" maxlength="64" required autocapitalize="none" spellcheck="false" /></label>
            <label v-if="selected === 'proxmox'">{{ t('admin.keyringPath') }}<input v-model="release.keyring" maxlength="150" required spellcheck="false" /></label>
          </div>
          <div v-if="selected === 'debian'" class="generator-flags">
            <label><input v-model="release.stable" type="checkbox" @change="!release.stable && (release.backports = false)" />{{ t('admin.releaseStable') }}</label>
            <label><input v-model="release.firmware" type="checkbox" />{{ t('admin.releaseFirmware') }}</label>
            <label><input v-model="release.backports" type="checkbox" :disabled="!release.stable" />{{ t('admin.releaseBackports') }}</label>
          </div>
          <div class="generator-actions"><button type="button" :disabled="index === 0" @click="moveRelease(index, -1)">{{ t('admin.moveUp') }}</button><button type="button" :disabled="index === draft.data.releases.length - 1" @click="moveRelease(index, 1)">{{ t('admin.moveDown') }}</button><button type="button" :disabled="draft.data.releases.length === 1" @click="removeRelease(index)">{{ t('admin.remove') }}</button></div>
        </article>
        <div class="generator-actions"><button type="button" :disabled="draft.data.releases.length >= 100" @click="addRelease">{{ t('admin.addRelease') }}</button><button type="submit" class="primary" :disabled="!valid">{{ t('admin.save') }}</button></div>
      </fieldset>
      <p v-if="!valid" class="generator-invalid" role="status">{{ t('admin.generatorInvalid') }}</p>
      <p class="generator-hint">{{ t('admin.generatorScope') }}</p>
      <details v-if="previewCatalog"><summary>{{ t('admin.preview') }}</summary><MirrorConfig :key="`${selected}:${draft.data.defaultRelease}`" :config="{ type: selected }" :catalog="previewCatalog" /></details>
    </form>
  </div>
</template>

<style scoped>
.generator-editor { max-width: 1050px; min-width: 0; }
.generator-editor > p { line-height: 1.8; }
fieldset { min-width: 0; margin: 20px 0; padding: 0; border: 0; }
label { display: grid; gap: 8px; min-width: 0; font-size: 14px; }
input:not([type='checkbox']), select { width: 100%; min-width: 0; min-height: 42px; padding: 9px 11px; border: 1px solid var(--line); border-radius: 3px; background: var(--panel); color: var(--ink); font: inherit; }
.generator-release { margin: 16px 0; padding: 18px; border: 1px solid var(--line); border-radius: 3px; }
.generator-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.generator-fields > label:nth-child(3) { grid-column: 1 / -1; }
.generator-flags, .generator-actions { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 16px; }
.generator-flags label { display: inline-flex; align-items: center; gap: 8px; }
button { min-height: 40px; padding: 8px 14px; border: 1px solid var(--line); border-radius: 3px; background: var(--panel); color: var(--ink); font: inherit; font-size: 14px; cursor: pointer; }
button:hover { border-color: var(--blue); color: var(--blue); }
button.primary { border-color: var(--blue); background: var(--blue); color: var(--on-accent); }
button:disabled { opacity: .5; cursor: default; }
.generator-hint { color: var(--ink-2); line-height: 1.8; font-size: 14px; }
.generator-invalid { color: var(--red, #c64040); line-height: 1.8; }
summary { color: var(--blue); cursor: pointer; }
@media (max-width: 640px) { .generator-fields { grid-template-columns: minmax(0, 1fr); } .generator-release { padding: 12px; } button { min-height: 44px; } }
</style>
