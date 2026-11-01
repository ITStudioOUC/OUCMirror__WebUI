<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SiteDownload } from '../api'
import MarkdownContent from './MarkdownContent.vue'

const props = defineProps<{ items: SiteDownload[] }>()
const { t } = useI18n()
const selectedId = ref('')
const selected = computed(() => props.items.find(item => item.id === selectedId.value))
function ensureSelection() {
  if (!props.items.some(item => item.id === selectedId.value)) selectedId.value = props.items[0]?.id || ''
}
watch(() => props.items, ensureSelection, { immediate: true })

async function moveProject(event: KeyboardEvent, index: number) {
  let next = index
  if (event.key === 'ArrowDown') next = (index + 1) % props.items.length
  else if (event.key === 'ArrowUp') next = (index + props.items.length - 1) % props.items.length
  else if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = props.items.length - 1
  else return
  event.preventDefault()
  selectedId.value = props.items[next]!.id
  await nextTick()
  document.getElementById(`download-project-${selectedId.value}`)?.focus()
}
</script>

<template>
  <div class="download-catalog">
    <div class="download-workspace">
      <div v-if="items.length" class="download-projects" role="tablist" aria-orientation="vertical" :aria-label="t('announcements.chooseSoftware')">
        <button v-for="(project, index) in items" :id="`download-project-${project.id}`" :key="project.id" role="tab" type="button" :aria-selected="selectedId === project.id" :tabindex="selectedId === project.id ? 0 : -1" aria-controls="download-project-panel" @click="selectedId = project.id" @keydown="moveProject($event, index)">{{ project.name }}</button>
      </div>
      <label v-if="items.length" class="download-mobile-select">{{ t('announcements.chooseSoftware') }}<select v-model="selectedId"><option v-for="project in items" :key="project.id" :value="project.id">{{ project.name }}</option></select></label>
      <section v-if="selected" id="download-project-panel" class="download-detail" role="tabpanel" :aria-labelledby="`download-project-${selected.id}`" tabindex="0">
        <h3>{{ selected.name }}</h3>
        <MarkdownContent v-if="selected.description" class="download-description" :source="selected.description" lang="zh-CN" />
        <ul class="download-files" :aria-label="t('announcements.downloadLinks')"><li v-for="(file, index) in selected.files" :key="`${index}:${file.url}`"><a :href="file.url" target="_blank" rel="noopener noreferrer">{{ file.label }}</a></li></ul>
      </section>
      <p v-else class="downloads-empty" role="status">{{ t('common.noDownloads') }}</p>
    </div>
  </div>
</template>

<style scoped>
.download-catalog { display: flex; flex-direction: column; flex: 1; min-width: 0; min-height: 0; }
.download-workspace { display: grid; grid-template-columns: 210px minmax(0, 1fr); flex: 1; min-height: 0; gap: 28px; }
.download-projects { min-height: 0; overflow: auto; overscroll-behavior: contain; padding-right: 6px; }
.download-projects button { display: block; width: 100%; min-height: 42px; padding: 10px 14px; border-radius: 2px; color: var(--blue); font-size: 15px; line-height: 1.5; text-align: left; overflow-wrap: anywhere; }
.download-projects button:hover { background: var(--blue-soft); }
.download-projects [aria-selected='true'] { background: var(--blue); color: var(--on-accent); font-weight: 600; }
.download-detail { min-width: 0; min-height: 0; overflow: auto; overscroll-behavior: contain; padding: 2px 8px 12px 0; overflow-wrap: anywhere; }
.download-detail h3 { margin: 0 0 18px; color: var(--ink); font-size: 28px; font-weight: 600; line-height: 1.4; }
.download-description { margin-bottom: 22px; font-size: 15px; }
.download-files { margin: 0; padding-left: 20px; }
.download-files li { padding: 5px 0; color: var(--ink); }
.download-files a { color: var(--blue); font-size: 16px; line-height: 1.6; text-underline-offset: 4px; }
.download-files a:hover { text-decoration: underline; }
.download-mobile-select { display: none; }
.downloads-empty { grid-column: 1 / -1; margin: 8px 0; color: var(--ink); font-size: 15px; }
@media (max-width: 640px) {
  .download-workspace { display: flex; flex-direction: column; gap: 20px; }
  .download-projects { display: none; }
  .download-mobile-select { display: grid; gap: 7px; flex-shrink: 0; color: var(--ink); font-size: 14px; }
  .download-mobile-select select { width: 100%; min-width: 0; min-height: 44px; padding: 8px 10px; border: 1px solid var(--line); border-radius: 2px; background: var(--paper); color: var(--ink); font: inherit; font-size: 16px; }
  .download-detail { flex: 1; }
  .download-detail h3 { font-size: 24px; }
  .download-files li { padding-block: 8px; }
}
</style>
