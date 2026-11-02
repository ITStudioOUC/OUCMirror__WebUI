<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowRight, ArrowUpRight, CircleHelp, Check,
  ExternalLink, Menu, Monitor, Moon,
  Search, Sun, X,
} from 'lucide-vue-next'
import RollingText from './components/RollingText.vue'
import LanguagePicker from './components/LanguagePicker.vue'
import StatusPage from './components/StatusPage.vue'
import AdminPage from './components/AdminPage.vue'
import AnnouncementsPage from './components/AnnouncementsPage.vue'
import HelpPage from './components/HelpPage.vue'
import DownloadList from './components/DownloadList.vue'
import { content } from './api'
import { announcementUrl } from './announcements'
import { helpRoute, helpUrl } from './help'
import { usePublicData } from './usePublicData'
import { useMirrorFormats } from './useMirrorFormats'
import { type Mirror } from './data'
import { datetimeFormats, readPreference, savePreference, type Locale } from './locales'

const { t, locale } = useI18n()
const isAdminPage = window.location.pathname.startsWith('/admin')
const { site, mirrors, announcements: articleRecords, source, loading } = usePublicData()
const announcements = computed(() => articleRecords.value.map(row => ({ id: row.id, ...row.data })))
const distributions = computed(() => site.value?.downloads || [])
const description = (mirror: Mirror) => content(mirror.description, locale.value)
const language = computed(() => locale.value as Locale)
type Theme = 'light' | 'dark' | 'system'
const storedTheme = readPreference('mirrorouc.theme')
const theme = ref<Theme>(storedTheme === 'light' || storedTheme === 'dark' ? storedTheme : 'system')
const systemDark = window.matchMedia('(prefers-color-scheme: dark)')
const prefersDark = ref(systemDark.matches)
const dark = computed(() => theme.value === 'dark' || (theme.value === 'system' && prefersDark.value))
const brandLogo = computed(() => dark.value ? site.value?.logoDark || '/assets/mirrorOUC.png' : site.value?.logoLight || '/assets/mirrorOUC_dark.png')
watch(() => site.value?.favicon, value => { if (value) document.querySelector('link[rel="icon"]')?.setAttribute('href', value) })
const themeOpen = ref(false)
const mobileOpen = ref(false)
const themeRoot = ref<HTMLElement>()
const themes: Theme[] = ['light', 'dark', 'system']
const themeIcons = { light: Sun, dark: Moon, system: Monitor }
const isStatusPage = window.location.pathname.replace(/\/$/, '') === '/status'
const isAnnouncementsPage = window.location.pathname.replace(/\/$/, '') === '/announcements'
const help = helpRoute(window.location.pathname)
const isHelpPage = help.active
const navigation = [
  { key: 'mirrors', href: '/', active: !isStatusPage && !isAdminPage && !isAnnouncementsPage },
  { key: 'news', href: '/announcements', active: isAnnouncementsPage },
  { key: 'clubNews', href: 'https://www.itouc.cn/news/', active: false },
  { key: 'status', href: '/status', active: isStatusPage },
  { key: 'feedback', href: 'https://git.itouc.cn/ITStudio_OUC/OUCMirror-WebUI/issues', active: false },
]

watch(dark, (value) => {
  document.documentElement.dataset.theme = value ? 'dark' : 'light'
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', value ? '#0d1117' : '#ffffff')
}, { immediate: true })
watch(theme, (value) => savePreference('mirrorouc.theme', value))
watch(locale, (value) => {
  document.documentElement.lang = value
  document.title = t('common.pageTitle')
  savePreference('mirrorouc.locale', value)
}, { immediate: true })
watch([locale, site], () => {
  const siteTitle = site.value ? `${t('common.brand')} · ${content(site.value.name, locale.value)}` : t('common.pageTitle')
  const pageTitle = isHelpPage ? (help.name && !help.invalid ? t('mirrors.helpFor', { name: help.name }) : t('mirrors.helpTitle')) : isAnnouncementsPage ? t('announcements.title') : ''
  document.title = pageTitle ? `${pageTitle} · ${siteTitle}` : siteTitle
}, { immediate: true })
function selectTheme(value: Theme) { theme.value = value; themeOpen.value = false }

const search = ref('')
const searchInput = ref<HTMLInputElement>()
const statusPage = ref<InstanceType<typeof StatusPage>>()
const filteredMirrors = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return mirrors.value
  return mirrors.value.filter(mirror => [mirror.name, ...Object.values(mirror.description || {})].join(' ').toLowerCase().includes(query))
})
const now = ref(Date.now())
const { mirrorSize, relativeTime, tableNextTime, fullTime } = useMirrorFormats(now)
const showNextSync = ref(false)
const boardCycle = ref(0)
const timeHeadings = computed(() => [t('mirrors.columns.last'), t('mirrors.columns.next')])
const metricHeadings = computed(() => [t('mirrors.columns.size'), t('mirrors.columns.tableStatus')])
const statusLabels = computed(() => ['success', 'syncing', 'waiting', 'failed', 'paused', 'unknown'].map(status => t(`mirrors.status.${status}`)))
const timeCandidates = computed(() => [...new Set(mirrors.value.flatMap(mirror => [relativeTime(mirror.lastSuccess), tableNextTime(mirror)]))])
const sizeCandidates = computed(() => mirrors.value.map(mirrorSize))
function announcementDate(date: string) {
  const formatter = new Intl.DateTimeFormat(language.value, datetimeFormats[language.value].announcement)
  const parts = Object.fromEntries(formatter.formatToParts(new Date(date)).map(part => [part.type, part.value]))
  return t('mirrors.datePatterns.announcement', { year: parts.year, month: parts.month, day: parts.day })
}
function nextLabel(mirror: Mirror) {
  return ['syncing', 'failed', 'paused', 'unknown'].includes(mirror.status)
    ? t(`mirrors.status.${mirror.status}`) : relativeTime(mirror.nextSync, true)
}
function resetSearch() { search.value = ''; searchInput.value?.focus() }

type Modal = 'about' | 'downloads' | 'mirror'
const dialog = ref<HTMLDialogElement>()
const modal = ref<Modal | null>(null)
const selectedMirror = ref<Mirror | null>(null)
const modalTitle = computed(() => {
  if (modal.value === 'mirror') return selectedMirror.value?.name || ''
  const keys = { about: 'common.aboutTitle', downloads: 'announcements.downloadsTitle' }
  return modal.value && modal.value in keys ? t(keys[modal.value as keyof typeof keys]) : ''
})
async function openModal(value: Modal, mirror?: Mirror) {
  modal.value = value
  selectedMirror.value = mirror || null
  mobileOpen.value = false
  themeOpen.value = false
  await nextTick()
  if (!dialog.value?.open) dialog.value?.showModal()
}
function closeModal() { dialog.value?.close() }
function onDialogClose() { modal.value = null; selectedMirror.value = null }
watch(modal, value => { document.body.classList.toggle('dialog-open', value !== null) })

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') { themeOpen.value = false; mobileOpen.value = false }
  const target = event.target as HTMLElement
  if (!isAdminPage && !isAnnouncementsPage && !isHelpPage && !dialog.value?.open && ((event.key === 'k' && (event.ctrlKey || event.metaKey))
    || (event.key === '/' && !['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) && !target.isContentEditable))) {
    event.preventDefault()
    if (isStatusPage) statusPage.value?.focusSearch()
    else searchInput.value?.focus()
  }
}
function outsideClick(event: MouseEvent) {
  if (!themeRoot.value?.contains(event.target as Node)) themeOpen.value = false
}
function onSystemTheme(event: MediaQueryListEvent) { prefersDark.value = event.matches }
let interval: ReturnType<typeof setInterval>
let boardInterval: ReturnType<typeof setInterval>
onMounted(() => {
  document.addEventListener('keydown', onKeydown)
  document.addEventListener('click', outsideClick)
  systemDark.addEventListener('change', onSystemTheme)
  boardInterval = setInterval(() => {
    if (document.hidden) return
    showNextSync.value = !showNextSync.value
    boardCycle.value++
  }, 5000)
  interval = setInterval(() => {
    if (document.visibilityState !== 'visible') return
    now.value = Date.now()
  }, 1000)
})
onBeforeUnmount(() => {
  clearInterval(interval)
  clearInterval(boardInterval)
  document.removeEventListener('keydown', onKeydown)
  document.removeEventListener('click', outsideClick)
  systemDark.removeEventListener('change', onSystemTheme)
  document.body.classList.remove('dialog-open')
})
</script>

<template>
  <a class="skip-link" :href="isAdminPage ? '#administration' : isStatusPage ? '#site-status' : isAnnouncementsPage ? '#announcements' : isHelpPage ? '#mirror-help' : '#mirrors'">{{ t(isAdminPage ? 'common.skipAdmin' : isStatusPage ? 'common.skipStatus' : isAnnouncementsPage ? 'common.skipAnnouncements' : isHelpPage ? 'common.skipHelp' : 'common.skip') }}</a>
  <header class="site-header">
      <a href="/" class="brand" :aria-label="content(site?.name, locale) || t('common.siteName')">
        <span class="brand-logo"><img :src="brandLogo" alt="" width="2000" height="433" /></span>
      </a>
      <nav class="desktop-nav" :aria-label="t('common.menu')">
        <a v-for="item in navigation" :key="item.key" class="nav-link" :class="{ active: item.active }" :href="item.href" :aria-current="item.active ? 'page' : undefined">{{ t(`common.nav.${item.key}`) }}</a>
      </nav>
      <div class="header-tools">
        <LanguagePicker @open="themeOpen = false" />
        <span class="tool-divider" />
        <div ref="themeRoot" class="theme-picker">
          <button class="icon-button" :aria-label="t('common.theme')" :aria-expanded="themeOpen" aria-controls="theme-options" @click="themeOpen = !themeOpen"><component :is="themeIcons[theme]" :size="19" /></button>
          <div v-if="themeOpen" id="theme-options" class="theme-options" role="group" :aria-label="t('common.theme')">
            <button v-for="choice in themes" :key="choice" :aria-pressed="theme === choice" @click="selectTheme(choice)"><component :is="themeIcons[choice]" :size="16" />{{ t(`common.themes.${choice}`) }}<Check v-if="theme === choice" :size="15" /></button>
          </div>
        </div>
        <button class="icon-button mobile-toggle" :aria-label="t('common.menu')" :aria-expanded="mobileOpen" aria-controls="mobile-nav" @click="mobileOpen = !mobileOpen"><X v-if="mobileOpen" :size="21" /><Menu v-else :size="21" /></button>
      </div>
    <nav v-if="mobileOpen" id="mobile-nav" class="mobile-nav" :aria-label="t('common.menu')">
      <a v-for="item in navigation" :key="item.key" :href="item.href" :aria-current="item.active ? 'page' : undefined" @click="mobileOpen = false">{{ t(`common.nav.${item.key}`) }}</a>
    </nav>
  </header>

  <main class="page-width">
    <p v-if="!isAdminPage && !isAnnouncementsPage && !isHelpPage && (loading || source.stale)" class="data-notice" role="status">{{ t(loading ? 'common.loading' : source.updatedAt ? 'common.stale' : 'common.unavailable') }}</p>
    <AdminPage v-if="isAdminPage" />
    <AnnouncementsPage v-else-if="isAnnouncementsPage" />
    <HelpPage v-else-if="isHelpPage" :name="help.name" :invalid="help.invalid" />
    <StatusPage v-else-if="isStatusPage" ref="statusPage" :mirrors="mirrors" :now="now" :cycle="boardCycle" :show-next-sync="showNextSync" @mirror="openModal('mirror', $event)" />
    <div v-else class="main-grid">
      <section id="mirrors" class="mirror-section" tabindex="-1">
        <div class="mirror-controls">
          <div class="section-heading"><h1>{{ t('mirrors.title') }}</h1></div>
          <div class="search-box"><Search :size="19" /><input ref="searchInput" v-model="search" type="search" :placeholder="t('mirrors.search')" :aria-label="t('mirrors.search')" autocomplete="off" /><button v-if="search" class="icon-button clear-search" :aria-label="t('mirrors.clear')" @click="search = ''"><X :size="16" /></button></div>
          <span class="sr-only" role="status">{{ t('mirrors.results', { count: filteredMirrors.length }) }}</span>
        </div>
        <div class="mirror-table-wrap">
          <table class="mirror-table" :data-time-field="showNextSync ? 'next' : 'last'" :data-metric-field="showNextSync ? 'status' : 'size'" :data-cycle="boardCycle">
            <thead><tr>
              <th scope="col">{{ t('mirrors.columns.name') }}</th>
              <th scope="col"><RollingText :value="timeHeadings[showNextSync ? 1 : 0]!" :cycle="boardCycle" :reserve="timeHeadings" :shuffle="false" /></th>
              <th scope="col"><RollingText :value="metricHeadings[showNextSync ? 1 : 0]!" :cycle="boardCycle" :reserve="metricHeadings" :shuffle="false" /></th>
            </tr></thead>
            <tbody>
              <tr v-for="mirror in filteredMirrors" :key="mirror.name" :data-status="mirror.status">
                <td class="name-cell">
                  <div class="mirror-name-group">
                    <a class="mirror-name-button" :aria-label="t('mirrors.browse', { name: mirror.name })" :title="description(mirror)" :href="mirror.browseUrl">{{ mirror.name }}</a>
                    <a v-if="mirror.help?.trim()" class="help-button" :href="helpUrl(mirror.name)" :aria-label="t('mirrors.helpFor', { name: mirror.name })" :title="t('mirrors.helpFor', { name: mirror.name })"><CircleHelp :size="22" :stroke-width="1.7" aria-hidden="true" /></a>
                  </div>
                </td>
                <td class="time-cell" :title="fullTime(showNextSync ? mirror.nextSync : mirror.lastSuccess)"><RollingText :value="showNextSync ? tableNextTime(mirror) : relativeTime(mirror.lastSuccess)" :cycle="boardCycle" :candidates="timeCandidates" :reserve="[relativeTime(mirror.lastSuccess), tableNextTime(mirror)]" /></td>
                <td class="metric-cell"><RollingText :value="showNextSync ? t(`mirrors.status.${mirror.status}`) : mirrorSize(mirror)" :cycle="boardCycle" :candidates="showNextSync ? statusLabels : sizeCandidates" :reserve="[mirrorSize(mirror), t(`mirrors.status.${mirror.status}`)]" /></td>
              </tr>
            </tbody>
          </table>
          <div v-if="!filteredMirrors.length" class="empty-state"><Search :size="30" :stroke-width="1.3" /><h3>{{ t('mirrors.empty') }}</h3><p>{{ t('mirrors.emptyHint') }}</p><button class="text-button" @click="resetSearch">{{ t('mirrors.reset') }}<ArrowRight :size="15" /></button></div>
        </div>
      </section>

      <aside class="sidebar">
        <section class="side-section announcements">
          <div class="side-heading"><h2>{{ t('announcements.title') }}</h2><a class="sidebar-more" href="/announcements" :aria-label="t('announcements.more')">{{ t('announcements.moreLabel') }}</a></div>
          <a v-for="item in announcements" :key="item.id" class="news-item" :href="announcementUrl(item.id)" :title="item.title" :aria-label="`${announcementDate(item.date)} ${item.title}`">
            <time :datetime="item.date">{{ announcementDate(item.date) }}</time>
            <span class="news-title" lang="zh-CN">{{ item.title }}</span>
          </a>
          <p v-if="!announcements.length" class="dialog-hint">{{ t('common.noAnnouncements') }}</p>
        </section>

        <section class="side-section downloads">
          <div class="side-heading"><h2>{{ t('announcements.downloads') }}</h2><button class="download-menu-button" aria-haspopup="dialog" aria-controls="content-dialog" @click="openModal('downloads')">{{ t('announcements.downloadMenu') }}</button></div>
        </section>

        <section v-if="site?.relatedLinks?.length" class="side-section useful-links">
          <div class="side-heading"><h2>{{ t('announcements.links') }}</h2></div>
          <template v-for="(link, index) in site.relatedLinks" :key="index">
            <a v-if="link.kind === 'help'" href="/help/">{{ content(link.title, locale) }}</a>
            <a v-else :href="link.url" :target="link.url.startsWith('/') ? undefined : '_blank'" rel="noopener noreferrer">{{ content(link.title, locale) }}</a>
          </template>
        </section>
      </aside>
    </div>
  </main>

  <div v-if="!isAdminPage" class="footer-affiliations">
    <a class="footer-affiliation footer-studio" href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer" :aria-label="t('common.studio')">
      <img src="/assets/itstudio.png" alt="" width="3849" height="990" />
    </a>
    <a class="footer-affiliation footer-university" href="https://www.ouc.edu.cn/" target="_blank" rel="noopener noreferrer" :aria-label="t('common.university')">
      <img src="/assets/ouc.png" alt="" width="4384" height="1106" />
    </a>
  </div>
  <footer v-if="!isAdminPage" class="site-footer">
    <div class="page-width">
      <div class="footer-main">
        <div class="footer-identity">
          <a href="/" class="footer-brand" :aria-label="t('common.siteName')">
            <img :src="brandLogo" alt="" width="2000" height="433" />
          </a>
          <p>{{ content(site?.description, locale) || t('common.footer') }}</p>
        </div>
        <nav class="footer-links" :aria-label="t('common.footerLinks')">
          <h2>{{ t('common.footerLinks') }}</h2>
          <ul v-if="!site">
            <li><a href="https://www.ouc.edu.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.university') }}</a></li>
            <li><a href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studio') }}</a></li>
            <li><a href="https://it.ouc.edu.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.faculty') }}</a></li>
            <li><a href="https://honor.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studioHonors') }}</a></li>
            <li><a href="https://git.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.gitPlatform') }}</a></li>
            <li><a href="https://blog.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studioBlog') }}</a></li>
          </ul>
          <ul v-else><li v-for="link in site.links" :key="link.url"><a :href="link.url" target="_blank" rel="noopener noreferrer">{{ content(link.title, locale) }}</a></li></ul>
        </nav>
        <section class="footer-contact">
          <h2>{{ t('common.footerContact') }}</h2>
          <a :href="`mailto:${site?.email || 'contact@itstudio.club'}`">{{ site?.email || t('common.contactEmail') }}</a>
          <address>{{ content(site?.address, locale) || t('common.studioAddress') }}</address>
        </section>
      </div>
    </div>
  </footer>

  <dialog id="content-dialog" ref="dialog" class="content-dialog" :class="{ 'download-dialog': modal === 'downloads' }" aria-labelledby="dialog-title" @close="onDialogClose" @click="($event.target === $event.currentTarget) && closeModal()">
    <div class="dialog-inner"><div class="dialog-heading"><h2 id="dialog-title">{{ modalTitle }}</h2><button class="icon-button" :aria-label="t('common.close')" autofocus @click="closeModal"><X :size="21" /></button></div>
      <div class="dialog-content">
        <template v-if="modal === 'mirror' && selectedMirror"><p class="dialog-lead">{{ description(selectedMirror) }}</p><div class="mirror-facts"><span>{{ t('mirrors.columns.status') }}</span><span>{{ t(`mirrors.status.${selectedMirror.status}`) }}</span><span>{{ t('mirrors.columns.updated') }}</span><span>{{ relativeTime(selectedMirror.lastSuccess) }}</span><span>{{ t('mirrors.columns.next') }}</span><span>{{ nextLabel(selectedMirror) }}</span></div><div class="dialog-actions"><a v-if="selectedMirror.help?.trim()" class="text-button" :href="helpUrl(selectedMirror.name)">{{ t('mirrors.helpTitle') }}<CircleHelp :size="18" /></a><a class="text-button" :href="selectedMirror.browseUrl">{{ t('common.openDirectory') }}<ArrowRight :size="15" /></a><a v-if="selectedMirror.website" :href="selectedMirror.website" target="_blank" rel="noopener noreferrer" class="text-button">{{ t('mirrors.official') }}<ExternalLink :size="14" /></a></div></template>
        <DownloadList v-else-if="modal === 'downloads'" :items="distributions" />
        <template v-else-if="modal === 'about'"><img class="about-seal" :src="site?.avatar || '/assets/ouc-seal.png'" alt="" width="80" height="80" /><p class="article-body">{{ t('common.aboutBody') }}</p><div class="about-links"><a href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studio') }}<ArrowUpRight :size="15" /></a><a href="https://github.com/tuna/tunasync" target="_blank" rel="noopener noreferrer">{{ t('common.source') }} · tunasync<ArrowUpRight :size="15" /></a></div></template>
      </div>
    </div>
  </dialog>
</template>
