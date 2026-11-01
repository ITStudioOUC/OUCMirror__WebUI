<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowRight, ArrowUpRight, BookOpen, Check,
  ExternalLink, Menu, Monitor, Moon,
  Search, Sun, X,
} from 'lucide-vue-next'
import RollingText from './components/RollingText.vue'
import LanguagePicker from './components/LanguagePicker.vue'
import StatusPage from './components/StatusPage.vue'
import { useMirrorFormats } from './useMirrorFormats'
import { announcements, demoMirrors, distributions, type Mirror } from './data'
import { datetimeFormats, readPreference, savePreference, supportedLocales, type Locale } from './locales'

const { t, d, locale } = useI18n()
const language = computed(() => locale.value as Locale)
const localeMessage = (key: string, selectedLocale: Locale = language.value) => t(key, {}, { locale: selectedLocale })
type Theme = 'light' | 'dark' | 'system'
const storedTheme = readPreference('mirrorouc.theme')
const theme = ref<Theme>(storedTheme === 'light' || storedTheme === 'dark' ? storedTheme : 'system')
const systemDark = window.matchMedia('(prefers-color-scheme: dark)')
const prefersDark = ref(systemDark.matches)
const dark = computed(() => theme.value === 'dark' || (theme.value === 'system' && prefersDark.value))
const brandLogo = computed(() => dark.value ? '/assets/mirrorOUC.png' : '/assets/mirrorOUC_dark.png')
const themeOpen = ref(false)
const mobileOpen = ref(false)
const themeRoot = ref<HTMLElement>()
const themes: Theme[] = ['light', 'dark', 'system']
const themeIcons = { light: Sun, dark: Moon, system: Monitor }
const isStatusPage = window.location.pathname.replace(/\/$/, '') === '/status'
const navigation = [
  { key: 'mirrors', href: '/', active: !isStatusPage },
  { key: 'news', href: 'https://www.itouc.cn/announcements/', active: false },
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
function selectTheme(value: Theme) { theme.value = value; themeOpen.value = false }

const search = ref('')
const searchInput = ref<HTMLInputElement>()
const statusPage = ref<InstanceType<typeof StatusPage>>()
const mirrors = ref(demoMirrors.map(mirror => ({ ...mirror })))
const filteredMirrors = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return mirrors.value
  return mirrors.value.filter(mirror => [mirror.name, ...supportedLocales.map(value => localeMessage(mirror.descriptionKey, value))].join(' ').toLowerCase().includes(query))
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
let demoStep = 0
function advanceDemo() {
  demoStep++
  const mirror = mirrors.value.find(item => item.name === (demoStep % 2 ? 'anaconda' : 'debian'))!
  mirror.status = mirror.status === 'syncing' ? 'success' : 'syncing'
  mirror.lastAttempt = Date.now()
  if (mirror.status === 'success') { mirror.lastSuccess = mirror.lastAttempt; mirror.nextSync = mirror.lastAttempt + 60 * 60_000; mirror.syncStarted = null }
  else { mirror.nextSync = null; mirror.syncStarted = mirror.lastAttempt }
}

type Modal = 'help' | 'news' | 'about' | 'downloads' | 'mirror'
const dialog = ref<HTMLDialogElement>()
const modal = ref<Modal | null>(null)
const selectedMirror = ref<Mirror | null>(null)
const selectedNews = ref<string | null>(null)
const copied = ref(false)
const copyError = ref(false)
const activeNews = computed(() => announcements.find(item => item.id === selectedNews.value))
const modalTitle = computed(() => {
  if (modal.value === 'mirror') return selectedMirror.value?.name || ''
  if (modal.value === 'news' && activeNews.value) return t(activeNews.value.titleKey)
  const keys = { help: 'mirrors.helpTitle', news: 'announcements.title', about: 'common.aboutTitle', downloads: 'announcements.downloadsTitle' }
  return modal.value && modal.value in keys ? t(keys[modal.value as keyof typeof keys]) : ''
})
async function openModal(value: Modal, mirror?: Mirror, news?: string) {
  modal.value = value
  selectedMirror.value = mirror || null
  selectedNews.value = news || null
  copied.value = false
  copyError.value = false
  mobileOpen.value = false
  themeOpen.value = false
  await nextTick()
  if (!dialog.value?.open) dialog.value?.showModal()
}
function closeModal() { dialog.value?.close() }
function onDialogClose() { modal.value = null; selectedMirror.value = null }
watch(modal, value => { document.body.classList.toggle('dialog-open', value !== null) })
const configExample = computed(() => {
  switch (selectedMirror.value?.name) {
    case 'ubuntu': return 'deb https://example.invalid/ubuntu/ noble main universe'
    case 'debian': return 'deb https://example.invalid/debian/ stable main'
    case 'pypi': return 'pip config set global.index-url https://example.invalid/pypi/simple'
    default: return `https://example.invalid/${selectedMirror.value?.name || ''}/`
  }
})
async function copyConfig() {
  copied.value = false
  copyError.value = false
  try { await navigator.clipboard.writeText(configExample.value); copied.value = true }
  catch { copyError.value = true }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') { themeOpen.value = false; mobileOpen.value = false }
  const target = event.target as HTMLElement
  if (!dialog.value?.open && ((event.key === 'k' && (event.ctrlKey || event.metaKey))
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
let elapsed = 0
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
    if (++elapsed % 9 === 0) advanceDemo()
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
  <a class="skip-link" :href="isStatusPage ? '#site-status' : '#mirrors'">{{ t(isStatusPage ? 'common.skipStatus' : 'common.skip') }}</a>
  <header class="site-header">
      <a href="/" class="brand" :aria-label="t('common.siteName')">
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
    <StatusPage v-if="isStatusPage" ref="statusPage" :mirrors="mirrors" :now="now" :cycle="boardCycle" :show-next-sync="showNextSync" @mirror="openModal('mirror', $event)" />
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
                    <button class="mirror-name-button" :aria-label="t('mirrors.browse', { name: mirror.name })" :title="t(mirror.descriptionKey)" @click="openModal('mirror', mirror)">{{ mirror.name }}</button>
                    <button class="help-button" :aria-label="t('mirrors.helpFor', { name: mirror.name })" :title="t('mirrors.helpFor', { name: mirror.name })" @click="openModal('mirror', mirror)"><BookOpen :size="14" /></button>
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
          <div class="side-heading"><h2>{{ t('announcements.title') }}</h2><button class="sidebar-more" :aria-label="t('announcements.more')" @click="openModal('news')">{{ t('announcements.moreLabel') }}</button></div>
          <button v-for="item in announcements" :key="item.id" class="news-item" :title="t(item.titleKey)" :aria-label="`${announcementDate(item.date)} ${t(item.titleKey)}`" @click="openModal('news', undefined, item.id)">
            <time :datetime="item.date">{{ announcementDate(item.date) }}</time>
            <span class="news-title">{{ t(item.titleKey) }}</span>
          </button>
        </section>

        <section class="side-section downloads">
          <div class="side-heading"><h2>{{ t('announcements.downloads') }}</h2><button class="download-menu-button" aria-haspopup="dialog" aria-controls="content-dialog" @click="openModal('downloads')">{{ t('announcements.downloadMenu') }}</button></div>
        </section>

        <section class="side-section useful-links">
          <div class="side-heading"><h2>{{ t('announcements.links') }}</h2></div>
          <button @click="openModal('help')">{{ t('announcements.linkHelp') }}</button>
          <a href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('announcements.linkStudio') }}</a>
          <a href="https://www.ouc.edu.cn/" target="_blank" rel="noopener noreferrer">{{ t('announcements.linkUniversity') }}</a>
        </section>
      </aside>
    </div>
  </main>

  <div class="footer-affiliations">
    <a class="footer-affiliation footer-studio" href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer" :aria-label="t('common.studio')">
      <img src="/assets/itstudio.png" alt="" width="3849" height="990" />
    </a>
    <a class="footer-affiliation footer-university" href="https://www.ouc.edu.cn/" target="_blank" rel="noopener noreferrer" :aria-label="t('common.university')">
      <img src="/assets/ouc.png" alt="" width="4384" height="1106" />
    </a>
  </div>
  <footer class="site-footer">
    <div class="page-width">
      <div class="footer-main">
        <div class="footer-identity">
          <a href="/" class="footer-brand" :aria-label="t('common.siteName')">
            <img :src="brandLogo" alt="" width="2000" height="433" />
          </a>
          <p>{{ t('common.footer') }}</p>
        </div>
        <nav class="footer-links" :aria-label="t('common.footerLinks')">
          <h2>{{ t('common.footerLinks') }}</h2>
          <ul>
            <li><a href="https://www.ouc.edu.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.university') }}</a></li>
            <li><a href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studio') }}</a></li>
            <li><a href="https://it.ouc.edu.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.faculty') }}</a></li>
            <li><a href="https://honor.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studioHonors') }}</a></li>
            <li><a href="https://git.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.gitPlatform') }}</a></li>
            <li><a href="https://blog.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studioBlog') }}</a></li>
          </ul>
        </nav>
        <section class="footer-contact">
          <h2>{{ t('common.footerContact') }}</h2>
          <a href="mailto:contact@itstudio.club">{{ t('common.contactEmail') }}</a>
          <address>{{ t('common.studioAddress') }}</address>
        </section>
      </div>
    </div>
  </footer>

  <dialog id="content-dialog" ref="dialog" class="content-dialog" aria-labelledby="dialog-title" @close="onDialogClose" @click="($event.target === $event.currentTarget) && closeModal()">
    <div class="dialog-inner"><div class="dialog-heading"><h2 id="dialog-title">{{ modalTitle }}</h2><button class="icon-button" :aria-label="t('common.close')" autofocus @click="closeModal"><X :size="21" /></button></div>
      <div class="dialog-content">
        <template v-if="modal === 'mirror' && selectedMirror"><p class="dialog-lead">{{ t(selectedMirror.descriptionKey) }}</p><div class="mirror-facts"><span>{{ t('mirrors.columns.status') }}</span><span class="status-chip" :class="selectedMirror.status"><span class="status-dot" />{{ t(`mirrors.status.${selectedMirror.status}`) }}</span><span>{{ t('mirrors.columns.updated') }}</span><span>{{ relativeTime(selectedMirror.lastSuccess) }}</span><span>{{ t('mirrors.columns.next') }}</span><span>{{ nextLabel(selectedMirror) }}</span></div><div class="directory-preview"><BookOpen :size="22" /><div><h3>{{ t('mirrors.directoryPending') }}</h3><p>{{ t('mirrors.directoryHint') }}</p></div></div><h3>{{ t('mirrors.exampleConfig') }}</h3><p class="dialog-hint">{{ t('mirrors.exampleHint') }}</p><pre><code>{{ configExample }}</code></pre><div class="dialog-actions"><button class="secondary-button" @click="copyConfig"><Check v-if="copied" :size="15" /><BookOpen v-else :size="15" />{{ copied ? t('common.copied') : t('common.copy') }}</button><a :href="selectedMirror.website" target="_blank" rel="noopener noreferrer" class="text-button">{{ t('mirrors.official') }}<ExternalLink :size="14" /></a></div><p v-if="copyError" role="status">{{ t('common.copyFailed') }}</p></template>
        <template v-else-if="modal === 'help'"><p class="dialog-lead">{{ t('mirrors.helpIntro') }}</p><div class="help-grid"><button v-for="mirror in mirrors" :key="mirror.name" @click="openModal('mirror', mirror)"><span class="distro-mark" :style="{ '--distro-color': mirror.color }" aria-hidden="true">{{ mirror.mark }}</span><span>{{ mirror.name }}</span><ArrowRight :size="15" /></button></div></template>
        <template v-else-if="modal === 'news'"><template v-if="activeNews"><time class="article-date" :datetime="activeNews.date">{{ d(new Date(activeNews.date), 'article') }}</time><p class="article-body">{{ t(activeNews.bodyKey) }}</p></template><template v-else><p class="dialog-lead">{{ t('announcements.newsIntro') }}</p><button v-for="item in announcements" :key="item.id" class="article-link" @click="selectedNews = item.id"><span>{{ t(item.titleKey) }}</span><ArrowRight :size="17" /></button></template></template>
        <template v-else-if="modal === 'downloads'"><p class="dialog-lead">{{ t('announcements.downloadsIntro') }}</p><div class="modal-downloads"><a v-for="distro in distributions" :key="distro.name" :href="distro.url" target="_blank" rel="noopener noreferrer"><span class="distro-mark" :style="{ '--distro-color': distro.color }" aria-hidden="true">{{ distro.mark }}</span><span>{{ distro.name }}</span><ArrowUpRight :size="18" /></a></div></template>
        <template v-else-if="modal === 'about'"><img class="about-seal" src="/assets/ouc-seal.png" alt="" width="80" height="80" /><p class="article-body">{{ t('common.aboutBody') }}</p><p class="dialog-hint">{{ t('common.previewBody') }}</p><div class="about-links"><a href="https://www.itouc.cn/" target="_blank" rel="noopener noreferrer">{{ t('common.studio') }}<ArrowUpRight :size="15" /></a><a href="https://github.com/tuna/tunasync" target="_blank" rel="noopener noreferrer">{{ t('common.source') }} · tunasync<ArrowUpRight :size="15" /></a></div></template>
      </div>
    </div>
  </dialog>
</template>
