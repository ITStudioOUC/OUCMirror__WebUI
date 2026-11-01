<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, ApiError, content, type HelpArticle, type HelpSummary } from '../api'
import { helpUrl } from '../help'
import MirrorHelpContent from './MirrorHelpContent.vue'

const props = defineProps<{ name: string | null; invalid: boolean }>()
const { t, locale } = useI18n()
const guides = ref<HelpSummary[]>([])
const article = ref<HelpArticle | null>(null)
const loading = ref(true)
const failed = ref(false)
const navigationFailed = ref(false)
const notFound = ref(props.invalid)
const title = computed(() => props.name && !props.invalid ? t('mirrors.helpFor', { name: props.name }) : t('mirrors.helpTitle'))
let controller: AbortController | undefined

async function load() {
  controller?.abort()
  const request = new AbortController()
  controller = request
  loading.value = true
  failed.value = false
  navigationFailed.value = false
  const [index, detail] = await Promise.allSettled([
    api<HelpSummary[]>('/help', { signal: request.signal }),
    props.name && !props.invalid ? api<HelpArticle>(`/help/${encodeURIComponent(props.name)}`, { signal: request.signal }) : Promise.resolve(null),
  ])
  if (request.signal.aborted) return
  if (index.status === 'fulfilled') guides.value = index.value
  else navigationFailed.value = true
  if (detail.status === 'fulfilled') { article.value = detail.value; notFound.value = props.invalid }
  else if (detail.reason instanceof ApiError && detail.reason.status === 404) { article.value = null; notFound.value = true }
  else failed.value = true
  if (!props.name && navigationFailed.value && !props.invalid) failed.value = true
  loading.value = false
}
function selectGuide(event: Event) {
  const name = (event.target as HTMLSelectElement).value
  window.location.assign(name ? helpUrl(name) : '/help/')
}
onMounted(() => { void load() })
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <section id="mirror-help" class="help-page" tabindex="-1" :aria-busy="loading">
    <div class="help-layout">
      <aside class="help-sidebar">
        <nav class="help-navigation" :aria-label="t('mirrors.guides.navigation')">
          <a href="/help/" :aria-current="!name && !invalid ? 'page' : undefined">{{ t('mirrors.helpTitle') }}</a>
          <a v-for="guide in guides" :key="guide.name" :href="helpUrl(guide.name)" :aria-current="guide.name === name ? 'page' : undefined">{{ guide.name }}</a>
        </nav>
        <label class="help-mobile-select">{{ t('mirrors.guides.choose') }}<select :value="name || ''" @change="selectGuide"><option value="">{{ t('mirrors.guides.all') }}</option><option v-if="name && !guides.some(guide => guide.name === name)" :value="name">{{ name }}</option><option v-for="guide in guides" :key="guide.name" :value="guide.name">{{ guide.name }}</option></select></label>
        <button v-if="navigationFailed && !failed" class="help-retry" @click="load">{{ t('mirrors.guides.retry') }}</button>
      </aside>
      <div class="help-content">
        <h1>{{ title }}</h1>
        <p v-if="loading" role="status">{{ t('mirrors.guides.loading') }}</p>
        <div v-else-if="failed" class="help-message" role="alert"><p>{{ t('mirrors.guides.failed') }}</p><button class="help-retry" @click="load">{{ t('mirrors.guides.retry') }}</button></div>
        <div v-else-if="notFound" class="help-message"><p>{{ t('mirrors.guides.notFound') }}</p><a href="/help/">{{ t('mirrors.guides.all') }}</a></div>
        <article v-else-if="article" class="help-article">
          <p v-if="content(article.description, locale)" class="help-description">{{ content(article.description, locale) }}</p>
          <div class="help-actions"><a :href="article.browseUrl">{{ t('common.openDirectory') }}</a><a v-if="article.website" :href="article.website" target="_blank" rel="noopener noreferrer">{{ t('mirrors.official') }}</a></div>
          <MirrorHelpContent :source="article.body" />
        </article>
        <div v-else class="help-index">
          <p>{{ t('mirrors.helpIntro') }}</p>
          <p v-if="!guides.length">{{ t('mirrors.guides.empty') }}</p>
          <ul v-else><li v-for="guide in guides" :key="guide.name"><a :href="helpUrl(guide.name)">{{ guide.name }}</a><span v-if="content(guide.description, locale)">{{ content(guide.description, locale) }}</span></li></ul>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.help-page { min-height: 50vh; padding: 32px 0 80px; }
.help-layout { display: grid; grid-template-columns: 210px minmax(0, 1fr); gap: 48px; align-items: start; }
.help-sidebar { position: sticky; top: 24px; min-width: 0; }
.help-navigation { display: grid; gap: 3px; max-height: calc(100dvh - 48px); overflow: auto; padding-right: 6px; }
.help-navigation a { display: block; padding: 10px 13px; color: var(--blue); font-size: 15px; line-height: 1.5; overflow-wrap: anywhere; }
.help-navigation a:hover { background: var(--blue-soft); }
.help-navigation [aria-current='page'] { background: var(--blue); color: var(--on-accent); font-weight: 600; border-radius: 2px; }
.help-content { min-width: 0; max-width: 940px; overflow-wrap: anywhere; }
h1 { margin: 0 0 20px; color: var(--ink); font-size: 36px; line-height: 1.35; font-weight: 600; }
.help-description { margin: 0 0 12px; color: var(--ink); font-size: 16px; line-height: 1.8; }
.help-actions { display: flex; flex-wrap: wrap; gap: 8px 24px; margin-bottom: 28px; }
.help-actions a, .help-message a, .help-index a { color: var(--blue); font-size: 15px; line-height: 1.8; text-underline-offset: 4px; }
.help-actions a:hover, .help-message a:hover, .help-index a:hover { text-decoration: underline; }
.help-article :deep(.markdown-content) { font-size: 16px; line-height: 1.9; }
.help-article :deep(.markdown-content h2) { margin-top: 1.6em; font-size: 26px; }
.help-article :deep(.markdown-content h3) { font-size: 20px; }
.help-article :deep(.markdown-content pre) { font-size: 13px; line-height: 1.8; }
.help-index p, .help-message p { margin: 0 0 22px; font-size: 16px; line-height: 1.9; }
.help-index ul { display: grid; gap: 16px; padding: 0; margin: 28px 0 0; list-style: none; }
.help-index li { display: grid; grid-template-columns: minmax(140px, 210px) minmax(0, 1fr); gap: 12px; align-items: baseline; font-size: 15px; line-height: 1.8; }
.help-index li > a { font-weight: 600; }
.help-retry { padding: 8px 14px; border: 1px solid var(--blue); border-radius: 3px; color: var(--blue); font-size: 14px; }
.help-mobile-select { display: none; }
@media (max-width: 760px) {
  .help-page { padding: 16px 0 48px; }
  .help-layout { grid-template-columns: minmax(0, 1fr); gap: 28px; }
  .help-sidebar { position: static; }
  .help-navigation { display: none; }
  .help-mobile-select { display: grid; gap: 8px; font-size: 14px; color: var(--ink); }
  .help-mobile-select select { width: 100%; min-width: 0; min-height: 44px; padding: 9px 12px; border: 1px solid var(--line); border-radius: 3px; background: var(--paper); color: var(--ink); font: inherit; font-size: 16px; }
  h1 { font-size: 28px; }
  .help-actions a, .help-message a { display: inline-flex; align-items: center; min-height: 44px; }
  .help-actions { margin-bottom: 18px; }
  .help-article :deep(.markdown-content h2) { font-size: 23px; }
  .help-index li { grid-template-columns: minmax(0, 1fr); gap: 0; }
}
</style>
