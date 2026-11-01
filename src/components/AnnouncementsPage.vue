<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, type AnnouncementContent, type ContentRecord } from '../api'
import { announcementAnchor, announcementUrl } from '../announcements'
import MarkdownContent from './MarkdownContent.vue'

const { t, d } = useI18n()
const articles = ref<ContentRecord<AnnouncementContent>[]>([])
const loading = ref(true)
const failed = ref(false)

async function locateArticle() {
  await nextTick()
  let anchor: string
  try { anchor = decodeURIComponent(window.location.hash.slice(1)) } catch { return }
  if (!articles.value.some(article => announcementAnchor(article.id) === anchor)) return
  const target = document.getElementById(anchor)
  target?.scrollIntoView({ block: 'start', behavior: 'instant' })
  target?.focus({ preventScroll: true })
}
async function load() {
  loading.value = true
  failed.value = false
  try {
    articles.value = await api('/announcements')
    await locateArticle()
  } catch { failed.value = true }
  finally { loading.value = false }
}
onMounted(() => {
  void load()
  window.addEventListener('hashchange', locateArticle)
})
onBeforeUnmount(() => window.removeEventListener('hashchange', locateArticle))
</script>

<template>
  <section id="announcements" class="announcements-page" tabindex="-1" :aria-busy="loading">
    <h1>{{ t('announcements.title') }}</h1>
    <p v-if="loading" role="status">{{ t('announcements.loading') }}</p>
    <div v-else-if="failed" class="announcement-error" role="alert">
      <p>{{ t('announcements.loadFailed') }}</p>
      <button @click="load">{{ t('announcements.retry') }}</button>
    </div>
    <p v-else-if="!articles.length">{{ t('common.noAnnouncements') }}</p>
    <article v-for="article in articles" :id="announcementAnchor(article.id)" :key="article.id" class="announcement" tabindex="-1" :aria-labelledby="`${announcementAnchor(article.id)}-title`">
      <h2 :id="`${announcementAnchor(article.id)}-title`"><a :href="announcementUrl(article.id)" lang="zh-CN">{{ article.data.title }}</a></h2>
      <div class="announcement-meta">
        <time :datetime="article.data.date">{{ d(new Date(article.data.date), 'article') }}</time>
        <span v-if="article.data.pinned">{{ t('announcements.pinned') }}</span>
      </div>
      <MarkdownContent :source="article.data.body" lang="zh-CN" />
    </article>
  </section>
</template>

<style scoped>
.announcements-page { max-width: 980px; min-height: 45vh; padding: 32px 0 80px; }
h1 { margin: 0 0 38px; color: var(--ink); font-size: 36px; font-weight: 600; line-height: 1.3; }
.announcement { margin: 0 0 48px; scroll-margin-top: 24px; overflow-wrap: anywhere; }
.announcement:last-child { margin-bottom: 0; }
h2 { margin: 0 0 10px; font-size: 26px; font-weight: 600; line-height: 1.5; }
h2 a { color: var(--blue); }
h2 a:hover { text-decoration: underline; text-underline-offset: 5px; }
.announcement-meta { display: flex; align-items: center; gap: 16px; margin-bottom: 22px; color: var(--muted); font-size: 14px; }
.announcement-meta span { color: var(--blue); }
.announcement :deep(.markdown-content) { font-size: 16px; line-height: 1.9; }
.announcement-error button { padding: 8px 16px; border: 1px solid var(--line); border-radius: 4px; color: var(--blue); }
:root[data-theme='dark'] h1 { color: #fff; }
@media (max-width: 640px) {
  .announcements-page { padding: 16px 0 48px; }
  h1 { margin-bottom: 28px; font-size: 28px; }
  h2 { font-size: 22px; }
  .announcement { margin-bottom: 36px; }
  .announcement-meta { margin-bottom: 16px; }
}
</style>
