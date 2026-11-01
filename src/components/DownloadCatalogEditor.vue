<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SiteDownload } from '../api'
import MarkdownContent from './MarkdownContent.vue'

const items = defineModel<SiteDownload[]>({ required: true })
defineProps<{ busy: boolean }>()
const { t } = useI18n()
const selectedId = ref('')
const selected = computed(() => items.value.find(item => item.id === selectedId.value))
const selectedIndex = computed(() => items.value.findIndex(item => item.id === selectedId.value))
watch(() => items.value.map(item => item.id), ids => {
  if (!ids.includes(selectedId.value)) selectedId.value = ids[0] || ''
}, { immediate: true })
function addProject() {
  const id = Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('')
  items.value.push({ id, name: '', category: 'software', description: '', files: [{ label: '', url: '' }] })
  selectedId.value = id
}
function moveProject(offset: number) {
  const from = selectedIndex.value, to = from + offset
  if (from < 0 || to < 0 || to >= items.value.length) return
  const [item] = items.value.splice(from, 1)
  if (item) items.value.splice(to, 0, item)
}
function removeProject() {
  const from = selectedIndex.value
  if (from < 0) return
  items.value.splice(from, 1)
  selectedId.value = items.value[Math.min(from, items.value.length - 1)]?.id || ''
}
function moveFile(index: number, offset: number) {
  const files = selected.value?.files, to = index + offset
  if (!files || to < 0 || to >= files.length) return
  const [file] = files.splice(index, 1)
  if (file) files.splice(to, 0, file)
}
const invalidProject = computed(() => items.value.find(item => !item.name.trim() || !item.files.length || item.files.some(file => !file.label.trim() || !file.url.trim())))
function validate() {
  if (invalidProject.value) selectedId.value = invalidProject.value.id
  return !invalidProject.value
}
defineExpose({ validate })
</script>

<template>
  <div class="catalog-editor">
    <div class="catalog-toolbar"><button type="button" :disabled="busy || items.length >= 100" @click="addProject">{{ t('admin.addDownloadProject') }}</button><span>{{ t('admin.downloadProjectCount', { count: items.length }) }}</span></div>
    <div v-if="items.length" class="catalog-layout">
      <nav class="catalog-projects" :aria-label="t('admin.downloadProjects')"><button v-for="(item, index) in items" :key="item.id" type="button" :disabled="busy" :aria-current="selectedId === item.id ? 'true' : undefined" @click="selectedId = item.id">{{ item.name || t('admin.unnamedDownload', { index: index + 1 }) }}</button></nav>
      <fieldset v-if="selected" :disabled="busy" class="catalog-fields">
        <div class="catalog-project-actions"><button type="button" :disabled="selectedIndex === 0" @click="moveProject(-1)">{{ t('admin.moveUp') }}</button><button type="button" :disabled="selectedIndex === items.length - 1" @click="moveProject(1)">{{ t('admin.moveDown') }}</button><button type="button" @click="removeProject">{{ t('admin.removeDownloadProject') }}</button></div>
        <label>{{ t('admin.softwareName') }}<input v-model="selected.name" maxlength="100" required /></label>
        <label>{{ t('admin.downloadDescription') }}<textarea v-model="selected.description" rows="6" maxlength="30000" lang="zh-CN" /></label>
        <details v-if="selected.description" class="catalog-preview"><summary>{{ t('admin.preview') }}</summary><MarkdownContent :source="selected.description" lang="zh-CN" /></details>
        <section class="catalog-files" aria-labelledby="catalog-file-title"><h3 id="catalog-file-title">{{ t('admin.downloadFiles') }}</h3>
          <div v-for="(file, index) in selected.files" :key="index" class="catalog-file" role="group" :aria-label="t('admin.downloadFileEntry', { index: index + 1 })">
            <label>{{ t('admin.downloadLabel') }}<input v-model="file.label" required maxlength="160" /></label>
            <label>{{ t('admin.url') }}<input v-model="file.url" required maxlength="2047" :placeholder="t('admin.downloadUrlPlaceholder')" /></label>
            <div class="catalog-file-actions"><button type="button" :disabled="index === 0" @click="moveFile(index, -1)">{{ t('admin.moveUp') }}</button><button type="button" :disabled="index === selected.files.length - 1" @click="moveFile(index, 1)">{{ t('admin.moveDown') }}</button><button type="button" @click="selected.files.splice(index, 1)">{{ t('admin.remove') }}</button></div>
          </div>
          <button type="button" :disabled="selected.files.length >= 100" @click="selected.files.push({ label: '', url: '' })">{{ t('admin.addDownloadFile') }}</button>
        </section>
      </fieldset>
    </div>
    <p v-else>{{ t('admin.noDownloadProjects') }}</p>
  </div>
</template>

<style scoped>
.catalog-editor { min-width: 0; }
.catalog-toolbar { display: flex; align-items: center; gap: 20px; margin-bottom: 22px; font-size: 14px; }
.catalog-layout { display: grid; grid-template-columns: 210px minmax(0, 1fr); gap: 30px; align-items: start; }
.catalog-projects { display: grid; max-height: 70vh; overflow: auto; gap: 4px; padding-right: 6px; }
.catalog-projects button { display: grid; width: 100%; gap: 4px; padding: 12px; border: 0; text-align: left; overflow-wrap: anywhere; }
.catalog-projects [aria-current] { background: var(--blue-soft); color: var(--blue); }
.catalog-fields { display: grid; min-width: 0; gap: 22px; margin: 0; padding: 0; border: 0; }
.catalog-project-actions, .catalog-file-actions { display: flex; gap: 8px; flex-wrap: wrap; }
label { display: grid; gap: 8px; min-width: 0; font-size: 14px; }
input, textarea { width: 100%; min-width: 0; min-height: 40px; padding: 9px 11px; border: 1px solid var(--line); border-radius: 3px; background: var(--panel); color: var(--ink); font: inherit; }
textarea { resize: vertical; line-height: 1.7; }
button { min-height: 38px; padding: 7px 12px; border: 1px solid var(--line); border-radius: 3px; background: var(--panel); color: var(--ink); font: inherit; font-size: 14px; }
button:hover { border-color: var(--blue); color: var(--blue); } button:disabled { opacity: .5; cursor: default; }
.catalog-preview summary { cursor: pointer; color: var(--blue); } .catalog-preview .markdown-content { margin-top: 16px; }
.catalog-files h3 { margin: 0 0 14px; font-size: 20px; }
.catalog-file { display: grid; gap: 14px; margin-bottom: 18px; padding: 18px 0; border-top: 1px solid var(--line); }
.catalog-file-actions { justify-content: end; }
@media (max-width: 760px) {
  .catalog-layout { grid-template-columns: minmax(0, 1fr); gap: 22px; }
  .catalog-projects { display: flex; max-height: none; padding: 0 0 8px; overflow-x: auto; }
  .catalog-projects button { flex: 0 0 150px; }
  button { min-height: 44px; }
}
</style>
