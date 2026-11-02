import { createI18n } from 'vue-i18n'
import { directoryMessages } from './messages'
import type { SiteSettings } from '../api'
import '../style.css'
import './style.css'

// Nginx has already rendered every file link. This module only enhances the
// existing HTML: it never requests, reconstructs or intercepts the directory.
type Locale = keyof typeof directoryMessages
type Theme = 'light' | 'dark' | 'system'
const locales = Object.keys(directoryMessages) as Locale[]
const read = (key: string) => { try { return localStorage.getItem(key) } catch { return null } }
const save = (key: string, value: string) => { try { localStorage.setItem(key, value) } catch { /* Optional preference. */ } }
const savedLocale = read('mirrorouc.locale')
const preferredLocale = navigator.languages.map(value => locales.find(locale => locale.toLowerCase() === value.toLowerCase()) ?? locales.find(locale => locale.split('-')[0] === value.split('-')[0])).find(Boolean)
let locale: Locale = locales.includes(savedLocale as Locale) ? savedLocale as Locale : preferredLocale || 'zh-CN'
const i18n = createI18n({ legacy: false, locale, fallbackLocale: 'zh-CN', messages: directoryMessages })
const t = (key: string, params: Record<string, string | number> = {}) => i18n.global.t(key, params)
const system = matchMedia('(prefers-color-scheme: dark)')
const savedTheme = read('mirrorouc.theme')
let theme: Theme = savedTheme === 'light' || savedTheme === 'dark' ? savedTheme : 'system'
let site: SiteSettings | undefined
const table = document.querySelector<HTMLTableElement>('#list')!
const segments = location.pathname.split('/').filter(Boolean).slice(1).map(segment => {
  try { return decodeURIComponent(segment) } catch { return segment }
})
const path = '/mirrors/' + segments.join('/') + '/'
const parent = table.tBodies[0]?.rows[0]?.querySelector<HTMLAnchorElement>('a[href^="../"]')
if (parent && segments.length <= 1) parent.href = '/'
const rows = [...table.tBodies[0]!.rows].filter(row => !row.contains(parent || null)).map(row => ({ row, name: row.querySelector('a')?.textContent?.toLocaleLowerCase() || '', directory: row.querySelector('a')?.getAttribute('href')?.split('?')[0]?.endsWith('/') || false }))
const input = document.querySelector<HTMLInputElement>('#directory-search')!
const pagination = document.querySelector<HTMLElement>('.directory-pagination')!
const previous = document.querySelector<HTMLButtonElement>('#directory-previous')!
const next = document.querySelector<HTMLButtonElement>('#directory-next')!
const empty = document.querySelector<HTMLElement>('#directory-empty')!
let matches = rows
let visible: typeof rows = []
let page = 1
const pageSize = 100
for (const { row } of rows) row.hidden = true

function renderPage() {
  for (const { row } of visible) row.hidden = true
  visible = matches.slice((page - 1) * pageSize, page * pageSize)
  for (const { row } of visible) row.hidden = false
  const pages = Math.max(1, Math.ceil(matches.length / pageSize))
  previous.disabled = page === 1
  next.disabled = page === pages
  pagination.hidden = matches.length === 0
  pagination.querySelector('div')!.hidden = pages <= 1
  empty.hidden = matches.length > 0
  empty.textContent = t(rows.length ? 'directory.noResults' : 'directory.empty')
  document.querySelector('#directory-page')!.textContent = t('directory.page', { page, pages })
  const folders = rows.filter(row => row.directory).length
  document.querySelector('#directory-summary')!.textContent = input.value.trim()
    ? t('directory.filtered', { count: matches.length })
    : t('directory.summary', { folders, files: rows.length - folders })
}

function safeURL(value: string) {
  try { const url = new URL(value, location.origin); return ['http:', 'https:'].includes(url.protocol) ? url.href : '' } catch { return '' }
}
function localized(value: Record<string, string> | undefined) { return value?.[locale] || value?.['zh-CN'] || Object.values(value || {}).find(Boolean) || '' }
function applySite() {
  const dark = document.documentElement.dataset.theme === 'dark'
  const logo = site && safeURL(dark ? site.logoDark : site.logoLight)
  for (const image of document.querySelectorAll<HTMLImageElement>('.directory-logo')) image.src = logo || (dark ? '/assets/mirrorOUC.png' : '/assets/mirrorOUC_dark.png')
  if (!site) return
  document.querySelector('#directory-description')!.textContent = localized(site.description) || t('common.footer')
  document.querySelector('#directory-address')!.textContent = localized(site.address) || t('common.studioAddress')
  const email = document.querySelector<HTMLAnchorElement>('#directory-email')!
  email.textContent = site.email || t('common.contactEmail')
  email.href = `mailto:${site.email || t('common.contactEmail')}`
  const favicon = safeURL(site.favicon)
  if (favicon) document.querySelector<HTMLLinkElement>('link[rel="icon"]')!.href = favicon
  const links = document.querySelector('#directory-footer-links')!
  links.replaceChildren(...(site.links || []).flatMap(link => {
    const url = safeURL(link.url)
    if (!url) return []
    const li = document.createElement('li'), a = document.createElement('a')
    a.href = url; a.textContent = localized(link.title); a.rel = 'noopener noreferrer'
    li.append(a)
    return [li]
  }))
}
function applyTheme() {
  const dark = theme === 'dark' || (theme === 'system' && system.matches)
  document.documentElement.dataset.theme = dark ? 'dark' : 'light'
  document.querySelector('meta[name="theme-color"]')!.setAttribute('content', dark ? '#0d1117' : '#ffffff')
  for (const button of document.querySelectorAll<HTMLButtonElement>('[data-theme-choice]')) button.setAttribute('aria-pressed', String(button.dataset.themeChoice === theme))
  applySite()
}
function applyLocale() {
  i18n.global.locale.value = locale
  document.documentElement.lang = locale
  document.title = `${path} · ${t('directory.title')} · ${t('common.brand')}`
  for (const element of document.querySelectorAll<HTMLElement>('[data-i18n]')) element.textContent = t(element.dataset.i18n!)
  for (const [attribute, data] of [['aria-label', 'i18nAria'], ['placeholder', 'i18nPlaceholder']] as const) {
    for (const element of document.querySelectorAll<HTMLElement>(`[data-${data === 'i18nAria' ? 'i18n-aria' : 'i18n-placeholder'}]`)) element.setAttribute(attribute, t(element.dataset[data]!))
  }
  document.querySelector('[data-current-language]')!.textContent = directoryMessages[locale].common.languageName
  for (const button of document.querySelectorAll<HTMLButtonElement>('[data-locale]')) button.setAttribute('aria-pressed', String(button.dataset.locale === locale))
  renderPage()
  applySite()
}

// Build breadcrumbs using text nodes, never HTML from a repository path.
const breadcrumbs = document.querySelector('.directory-breadcrumbs')!
segments.forEach((segment, index) => {
  const slash = document.createElement('span'); slash.textContent = '/'; slash.setAttribute('aria-hidden', 'true')
  const a = document.createElement('a'); a.textContent = segment
  a.href = '/mirrors/' + segments.slice(0, index + 1).map(encodeURIComponent).join('/') + '/'
  if (index === segments.length - 1) a.setAttribute('aria-current', 'page')
  breadcrumbs.append(slash, a)
})
let searchTimer: ReturnType<typeof setTimeout>
input.addEventListener('input', () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { const query = input.value.trim().toLocaleLowerCase(); matches = rows.filter(row => row.name.includes(query)); page = 1; renderPage() }, 120)
})
previous.addEventListener('click', () => { page--; renderPage(); table.scrollIntoView({ block: 'start' }) })
next.addEventListener('click', () => { page++; renderPage(); table.scrollIntoView({ block: 'start' }) })
for (const button of document.querySelectorAll<HTMLButtonElement>('[data-locale]')) button.addEventListener('click', () => {
  locale = button.dataset.locale as Locale; save('mirrorouc.locale', locale); applyLocale(); button.closest('details')!.open = false
})
for (const button of document.querySelectorAll<HTMLButtonElement>('[data-theme-choice]')) button.addEventListener('click', () => {
  theme = button.dataset.themeChoice as Theme; save('mirrorouc.theme', theme); applyTheme(); button.closest('details')!.open = false
})
const details = [...document.querySelectorAll<HTMLDetailsElement>('details')]
for (const detail of details) detail.addEventListener('toggle', () => { if (detail.open) for (const other of details) if (other !== detail) other.open = false })
document.addEventListener('click', event => { for (const detail of details) if (!detail.contains(event.target as Node)) detail.open = false })
document.addEventListener('keydown', event => {
  if (event.key === 'Escape') for (const detail of details) { if (detail.open) { detail.open = false; detail.querySelector('summary')?.focus() } }
  const target = event.target as HTMLElement
  if ((event.key === '/' && !['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) && !target.isContentEditable) || (event.key === 'k' && (event.ctrlKey || event.metaKey))) { event.preventDefault(); input.focus() }
})
system.addEventListener('change', applyTheme)
document.body.classList.add('directory-enhanced')
applyTheme(); applyLocale()
// Optional branding/settings only. A failed Go API cannot prevent browsing.
fetch('/api/v1/public', { signal: AbortSignal.timeout(3000) }).then(response => {
  if (!response.ok) throw new Error('Settings unavailable')
  return response.json()
}).then(data => { site = data.site; applySite() }).catch(() => {})
