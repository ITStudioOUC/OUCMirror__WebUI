<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, ApiError, jsonBody, setSession, type AdminSession, type AdminUser, type AnnouncementContent, type AuditPage, type ConfigVersion, type ContentRecord, type MirrorMetadata, type Overview, type SiteSettings, type SyncConfig, type SyncConfigMirror, type SyncJob } from '../api'
import { supportedLocales } from '../locales'
import { announcementUrl } from '../announcements'
import MarkdownContent from './MarkdownContent.vue'

const { t, te, d, n } = useI18n({ useScope: 'global' })
const session = ref<AdminSession | null>(null)
const checking = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const tab = ref('overview')
const tabs = ['overview', 'mirrors', 'announcements', 'config', 'settings', 'users', 'audit']
const allowedTabs = computed(() => session.value?.role === 'admin' ? tabs : ['announcements', 'users'])
const credentials = reactive({ username: '', password: '' })
const overview = ref<Overview | null>(null)
const articles = ref<ContentRecord<AnnouncementContent>[]>([])
const metadata = ref<ContentRecord<MirrorMetadata>[]>([])
const settings = ref<ContentRecord<SiteSettings> | null>(null)
const users = ref<AdminUser[]>([])
const audit = ref<AuditPage>({ items: [], page: 1, pageSize: 10, total: 0, totalPages: 1 })
const search = ref('')
const jobs = computed(() => (overview.value?.jobs || []).filter(job => `${job.name} ${job.upstream}`.toLowerCase().includes(search.value.toLowerCase())).sort((a, b) => a.name.localeCompare(b.name, 'en')))
const articlesEditor = ref<ContentRecord<AnnouncementContent> | null>(null)
const articleForm = ref<HTMLFormElement>()
const articleView = ref<'write' | 'preview'>('write')
const articlePublished = computed(() => articlesEditor.value?.data.published ?? false)
const sortedArticles = computed(() => [...articles.value].sort((a, b) => Number(b.data.pinned) - Number(a.data.pinned) || Date.parse(b.data.date) - Date.parse(a.data.date)))
const articleDate = computed(() => articlesEditor.value?.data.date ? new Intl.DateTimeFormat('en-CA', { year: 'numeric', month: '2-digit', day: '2-digit', timeZone: 'Asia/Shanghai' }).format(new Date(articlesEditor.value.data.date)) : '')
const mirrorEditor = ref<ContentRecord<MirrorMetadata> | null>(null)
const editLocale = ref('zh-CN')
const logs = ref<{ name: string; text: string } | null>(null)
const config = ref<SyncConfig | null>(null)
const configScope = ref('global')
const selectedConfig = ref('')
const mirrorConfig = ref<SyncConfigMirror | null>(null)
const draft = ref<ConfigVersion | null>(null)
const versions = ref<ConfigVersion[]>([])
const newUser = reactive({ username: '', password: '', role: 'editor' })
const passwords = reactive({ currentPassword: '', newPassword: '', confirm: '' })
const confirmation = ref<{ message: string; action: () => Promise<void> } | null>(null)
const confirmDialog = ref<HTMLDialogElement>()
const syncFields = ['interval', 'retry', 'timeout'] as const
const imageFields = ['logoLight', 'logoDark', 'favicon', 'avatar'] as const
const localizedFields = ['name', 'description', 'address'] as const
const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value))
function date(value: number | string) { return d(new Date(value), 'monitorSample') }
function message(code: string) { return te(`admin.errors.${code}`) ? t(`admin.errors.${code}`) : t('admin.errors.INTERNAL_ERROR') }
function eventLabel(value: string) { return te(`admin.events.${value}`) ? t(`admin.events.${value}`) : te(`admin.errors.${value}`) ? t(`admin.errors.${value}`) : value }
function jobStatus(value: string) { return ({ 'pre-syncing': 'waiting', disabled: 'paused', none: 'unknown' } as Record<string, string>)[value] || value }
function jobSize(value: string) {
  const match = /^\s*(\d+(?:\.\d+)?)\s*([KMGTPE]?)(?:i?B)?\s*$/i.exec(value)
  if (!match) return t('common.emptyValue')
  const bytes = Number(match[1]) * 1024 ** (match[2] ? 'KMGTPE'.indexOf(match[2].toUpperCase()) + 1 : 0)
  const large = bytes >= 1e12
  return t(large ? 'mirrors.size.terabyte' : 'mirrors.size.gigabyte', { value: n(bytes / (large ? 1e12 : 1e9), 'size') })
}
async function run(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try { await action() }
  catch (e) {
    if (e instanceof ApiError && e.status === 401) { session.value = null; setSession(null) }
    error.value = message(e instanceof ApiError ? e.code : 'INTERNAL_ERROR')
  } finally { busy.value = false }
}
async function loadOverview() { overview.value = await api<Overview>('/admin/overview') }
async function loadAudit(page = 1) { audit.value = await api<AuditPage>(`/admin/audit?page=${page}`) }
function moveRelatedLink(index: number, offset: number) {
  const links = settings.value?.data.relatedLinks
  const destination = index + offset
  if (!links || destination < 0 || destination >= links.length) return
  const [entry] = links.splice(index, 1)
  if (entry) links.splice(destination, 0, entry)
}
async function loadTab(value: string) {
  if (value === 'overview' || value === 'mirrors') await loadOverview()
  if (value === 'mirrors') metadata.value = await api('/admin/records/mirrors')
  if (value === 'announcements') articles.value = await api('/admin/records/announcements')
  if (value === 'settings') settings.value = (await api<ContentRecord<SiteSettings>[]>('/admin/records/settings'))[0] || null
  if (value === 'config') { config.value = await api('/admin/config'); versions.value = await api('/admin/config/versions'); draft.value = null; selectedConfig.value = ''; mirrorConfig.value = null }
  if (value === 'users' && session.value?.role === 'admin') users.value = await api('/admin/users')
  if (value === 'audit') await loadAudit()
}
function changeTab(value: string) { void run(async () => { tab.value = value; await loadTab(value) }) }
async function login() {
  session.value = await api('/auth/login', { method: 'POST', body: jsonBody(credentials) }); setSession(session.value)
  credentials.password = ''; tab.value = session.value?.role === 'admin' ? 'overview' : 'announcements'
  await loadTab(tab.value)
}
async function logout() { await api('/auth/logout', { method: 'POST' }); session.value = null; setSession(null); mirrorEditor.value = null; articlesEditor.value = null; logs.value = null }
async function saveRecord<T>(kind: string, value: ContentRecord<T>) {
  const result = await api<ContentRecord<T>>(`/admin/records/${kind}/${encodeURIComponent(value.id)}`, { method: 'PUT', body: jsonBody({ data: value.data, revision: value.revision }) })
  notice.value = t('admin.saved'); return result
}
async function ask(message: string, action: () => Promise<void>) {
  confirmation.value = { message, action }
  await nextTick()
  confirmDialog.value?.showModal()
}
function confirmed() { const action = confirmation.value?.action; confirmDialog.value?.close(); if (action) void run(action) }
function newArticle() {
  const id = Array.from(crypto.getRandomValues(new Uint8Array(16)), byte => byte.toString(16).padStart(2, '0')).join('')
  articlesEditor.value = { id, revision: 0, updatedAt: 0, data: { title: '', body: '', published: false, pinned: false, date: new Date().toISOString() } }
  void focusArticle()
}
async function focusArticle() {
  articleView.value = 'write'
  await nextTick()
  articleForm.value?.scrollIntoView({ block: 'start' })
  articleForm.value?.querySelector<HTMLInputElement>('[name="article-title"]')?.focus({ preventScroll: true })
}
function editArticle(record: ContentRecord<AnnouncementContent>) { articlesEditor.value = clone(record); articlesEditor.value.data.title ||= ''; articlesEditor.value.data.body ||= ''; void focusArticle() }
function submitArticle(event: Event) {
  const published = ((event as SubmitEvent).submitter as HTMLButtonElement | null)?.value === 'publish'
  void run(() => saveArticle(published))
}
async function saveArticle(published: boolean) {
  if (!articlesEditor.value) return
  const record = clone(articlesEditor.value)
  const wasPublished = record.data.published
  record.data.published = published
  const saved = await saveRecord('announcements', record)
  articlesEditor.value = saved
  articles.value = [saved, ...articles.value.filter(row => row.id !== saved.id)]
  notice.value = t(published ? wasPublished ? 'admin.articleUpdated' : 'admin.articlePublished' : wasPublished ? 'admin.articleWithdrawn' : 'admin.draftSaved')
}
function deleteArticle(record: ContentRecord<AnnouncementContent>) {
  void ask(t('admin.deleteConfirm', { name: record.data.title }), async () => {
    await api(`/admin/records/announcements/${record.id}`, { method: 'DELETE', body: jsonBody({ revision: record.revision }) })
    articles.value = await api('/admin/records/announcements'); if (articlesEditor.value?.id === record.id) articlesEditor.value = null
  })
}
function editMirror(job: SyncJob) {
  mirrorEditor.value = clone(metadata.value.find(row => row.id === job.name) || { id: job.name, revision: 0, updatedAt: 0, data: { name: job.name, description: { 'zh-CN': '', 'en-US': '' }, help: '', category: 'os', visible: true, website: '' } })
  mirrorEditor.value.data.description ||= {}; mirrorEditor.value.data.help ||= ''
}
async function saveMirror() { if (!mirrorEditor.value) return; mirrorEditor.value = await saveRecord('mirrors', mirrorEditor.value); metadata.value = await api('/admin/records/mirrors') }
function command(job: SyncJob, action: string) {
  void ask(t('admin.commandConfirm', { name: job.name, action: t(`admin.${action}`) }), async () => {
    await api(`/admin/jobs/${encodeURIComponent(job.name)}/${action}`, { method: 'POST' }); notice.value = t('admin.submitted'); await loadOverview()
  })
}
async function showLogs(name: string) { const data = await api<{ content: string }>(`/admin/jobs/${encodeURIComponent(name)}/logs`); logs.value = { name, text: data.content } }
async function upload(event: Event, field: typeof imageFields[number]) {
  const input = event.target as HTMLInputElement, file = input.files?.[0]
  if (!file || !settings.value) return
  const form = new FormData(); form.append('file', file)
  const result = await api<{ url: string }>('/admin/uploads', { method: 'POST', body: form }); settings.value.data[field] = result.url; input.value = ''
}
function chooseMirrorConfig() {
  draft.value = null
  const entry = config.value?.mirrors.find(row => row.name === selectedConfig.value)
  mirrorConfig.value = entry ? clone(entry) : config.value ? { name: '', provider: 'rsync', upstream: '', interval: config.value.global.interval, retry: config.value.global.retry, timeout: config.value.global.timeout } : null
}
async function previewConfig() {
  if (!config.value) return
  const body = configScope.value === 'global' ? { baseHash: config.value.baseHash, global: config.value.global } : { baseHash: config.value.baseHash, mirror: mirrorConfig.value }
  draft.value = await api('/admin/config/preview', { method: 'POST', body: jsonBody(body) }); versions.value = await api('/admin/config/versions')
}
function applyConfig() { if (!draft.value) return; const id = draft.value.id; void ask(t('admin.applyConfirm'), async () => { await api('/admin/config/apply', { method: 'POST', body: jsonBody({ id }) }); await loadTab('config'); notice.value = t('admin.submitted') }) }
function rollback(id: string) { void ask(t('admin.rollbackConfirm'), async () => { await api('/admin/config/rollback', { method: 'POST', body: jsonBody({ id }) }); await loadTab('config'); notice.value = t('admin.submitted') }) }
async function createUser() { await api('/admin/users', { method: 'POST', body: jsonBody(newUser) }); newUser.password = ''; newUser.username = ''; users.value = await api('/admin/users'); notice.value = t('admin.saved') }
async function changePassword() {
  if (passwords.newPassword !== passwords.confirm) { error.value = t('admin.passwordMismatch'); return }
  await api('/auth/password', { method: 'PUT', body: jsonBody({ current: passwords.currentPassword, password: passwords.newPassword }) })
  Object.assign(passwords, { currentPassword: '', newPassword: '', confirm: '' }); setSession(null); session.value = null; notice.value = t('admin.saved')
}
let poll: ReturnType<typeof setInterval>
onMounted(async () => {
  try { session.value = await api('/auth/session'); setSession(session.value); tab.value = session.value?.role === 'admin' ? 'overview' : 'announcements'; await run(() => loadTab(tab.value)) }
  catch (e) { if (!(e instanceof ApiError && e.status === 401)) error.value = message(e instanceof ApiError ? e.code : 'INTERNAL_ERROR') }
  finally { checking.value = false }
  poll = setInterval(() => { if (session.value?.role === 'admin' && ['overview', 'mirrors'].includes(tab.value) && !document.hidden && !busy.value) void loadOverview().catch(() => {}) }, 15000)
})
onBeforeUnmount(() => clearInterval(poll))
</script>

<template>
  <section id="administration" class="admin-page" tabindex="-1" :aria-busy="busy || checking">
    <div class="admin-top"><h1>{{ t('admin.title') }}</h1><div v-if="session" class="admin-actions"><span>{{ session.username }}</span><button @click="run(logout)" :disabled="busy">{{ t('admin.logout') }}</button></div></div>
    <p v-if="checking">{{ t('admin.checking') }}</p>
    <p v-if="error" role="alert" class="admin-error">{{ error }}</p><p v-if="notice" role="status" class="admin-notice">{{ notice }}</p>
    <form v-if="!checking && !session" class="admin-login admin-form" @submit.prevent="run(login)">
      <label>{{ t('admin.username') }}<input v-model="credentials.username" name="username" autocomplete="username" required maxlength="128" /></label>
      <label>{{ t('admin.password') }}<input v-model="credentials.password" name="password" type="password" autocomplete="current-password" required maxlength="256" /></label>
      <button class="primary" :disabled="busy">{{ t(busy ? 'admin.busy' : 'admin.login') }}</button>
    </form>
    <template v-if="session">
      <nav class="admin-tabs" :aria-label="t('admin.title')"><button v-for="item in allowedTabs" :key="item" :aria-current="tab === item ? 'page' : undefined" :disabled="busy" @click="changeTab(item)">{{ t(`admin.${item}`) }}</button></nav>
      <div class="admin-toolbar"><h2>{{ t(`admin.${tab}`) }}</h2><button :disabled="busy" @click="run(() => loadTab(tab))">{{ t('admin.refresh') }}</button></div>
      <template v-if="tab === 'overview' && overview">
        <p v-if="overview.code" class="admin-error">{{ t('admin.sourceError') }}</p>
        <dl class="admin-summary"><div><dt>{{ t('admin.worker') }}</dt><dd>{{ overview.worker }}</dd></div><div><dt>{{ t('admin.total') }}</dt><dd>{{ overview.jobs.length }}</dd></div><div><dt>{{ t('admin.active') }}</dt><dd>{{ overview.jobs.filter(row => row.status === 'syncing').length }}</dd></div><div><dt>{{ t('admin.failed') }}</dt><dd>{{ overview.jobs.filter(row => row.status === 'failed').length }}</dd></div><div><dt>{{ t('admin.updated') }}</dt><dd>{{ overview.updatedAt ? date(overview.updatedAt) : t('admin.empty') }}</dd></div></dl>
      </template>
      <template v-if="tab === 'mirrors'">
        <input v-model="search" class="admin-search" type="search" :placeholder="t('admin.search')" :aria-label="t('admin.search')" />
        <div class="admin-table-wrap jobs-table"><table><thead><tr><th>{{ t('admin.name') }}</th><th>{{ t('admin.status') }}</th><th>{{ t('admin.size') }}</th><th>{{ t('admin.actions') }}</th></tr></thead><tbody><tr v-for="job in jobs" :key="job.name" :data-status="jobStatus(job.status)"><th>{{ job.name }}</th><td>{{ t(`mirrors.status.${jobStatus(job.status)}`) }}</td><td>{{ jobSize(job.size) }}</td><td><div class="row-actions"><button @click="editMirror(job)">{{ t('admin.edit') }}</button><button :disabled="busy" @click="run(() => showLogs(job.name))">{{ t('admin.logs') }}</button><button v-for="action in ['start','stop','restart','disable']" :key="action" :disabled="busy" @click="command(job, action)">{{ t(`admin.${action}`) }}</button></div></td></tr></tbody></table></div>
        <form v-if="mirrorEditor" class="admin-editor admin-form" @submit.prevent="run(saveMirror)"><div class="admin-toolbar"><h3>{{ mirrorEditor.id }} · {{ t('admin.metadata') }}</h3><button type="button" @click="mirrorEditor = null">{{ t('admin.close') }}</button></div><label>{{ t('admin.descriptionLanguage') }}<select v-model="editLocale"><option v-for="language in supportedLocales" :key="language" :value="language">{{ t('common.languageName', {}, { locale: language }) }}</option></select></label><label>{{ t('admin.description') }}<textarea v-model="mirrorEditor.data.description[editLocale]" rows="2" /></label><label>{{ t('admin.helpChinese') }}<textarea v-model="mirrorEditor.data.help" lang="zh-CN" rows="10" /></label><MarkdownContent :source="mirrorEditor.data.help" lang="zh-CN" /><label>{{ t('admin.website') }}<input v-model="mirrorEditor.data.website" type="url" /></label><label>{{ t('admin.category') }}<select v-model="mirrorEditor.data.category"><option v-for="value in ['os','language','other']" :key="value" :value="value">{{ t(`admin.${value}`) }}</option></select></label><label class="check"><input v-model="mirrorEditor.data.visible" type="checkbox" />{{ t('admin.visible') }}</label><button class="primary" :disabled="busy">{{ t('admin.save') }}</button></form>
        <section v-if="logs" class="admin-editor"><div class="admin-toolbar"><h3>{{ logs.name }} · {{ t('admin.logs') }}</h3><div class="admin-actions"><button :disabled="busy" @click="run(() => showLogs(logs!.name))">{{ t('admin.refresh') }}</button><button @click="logs = null">{{ t('admin.close') }}</button></div></div><p>{{ t('admin.logTail') }}</p><pre class="admin-log">{{ logs.text }}</pre></section>
      </template>
      <template v-if="tab === 'announcements'">
        <div class="admin-actions"><button class="primary" :disabled="busy" @click="newArticle">{{ t('admin.newArticle') }}</button><a class="admin-public-link" href="/announcements" target="_blank" rel="noopener noreferrer">{{ t('admin.viewAnnouncements') }}</a></div>
        <form v-if="articlesEditor" ref="articleForm" class="admin-editor admin-form announcement-editor" @submit.prevent="submitArticle">
          <div class="admin-toolbar"><h3>{{ t(articlesEditor.revision ? 'admin.editArticle' : 'admin.newArticle') }}</h3><button type="button" :disabled="busy" @click="articlesEditor = null">{{ t('admin.close') }}</button></div>
          <fieldset :disabled="busy">
            <label>{{ t('admin.articleTitle') }}<input v-model="articlesEditor.data.title" name="article-title" maxlength="300" lang="zh-CN" required /></label>
            <div class="article-options"><label>{{ t('admin.date') }}<input :value="articleDate" type="date" required @input="articlesEditor.data.date = ($event.target as HTMLInputElement).value ? `${($event.target as HTMLInputElement).value}T00:00:00+08:00` : ''" /></label><label class="check"><input v-model="articlesEditor.data.pinned" type="checkbox" />{{ t('admin.pinned') }}</label><span class="article-state">{{ t(articlePublished ? 'admin.published' : 'admin.draft') }}</span></div>
            <div class="article-view-switch" :aria-label="t('admin.articleView')"><button type="button" :aria-pressed="articleView === 'write'" @click="articleView = 'write'">{{ t('admin.write') }}</button><button type="button" :aria-pressed="articleView === 'preview'" @click="articleView = 'preview'">{{ t('admin.preview') }}</button></div>
            <div class="article-workspace" :data-view="articleView">
              <label class="article-source">{{ t('admin.body') }}<textarea v-model="articlesEditor.data.body" lang="zh-CN" rows="18" maxlength="250000" spellcheck="false" /></label>
              <section class="article-preview" :aria-label="t('admin.preview')"><h4>{{ t('admin.preview') }}</h4><h2 lang="zh-CN">{{ articlesEditor.data.title || t('admin.articleTitle') }}</h2><time v-if="articlesEditor.data.date" :datetime="articlesEditor.data.date">{{ d(new Date(articlesEditor.data.date), 'article') }}</time><MarkdownContent v-if="articlesEditor.data.body" :source="articlesEditor.data.body" lang="zh-CN" /><p v-else class="article-empty">{{ t('admin.previewEmpty') }}</p></section>
            </div>
            <div class="admin-actions article-submit"><button type="submit" value="publish" class="primary">{{ t(articlePublished ? 'admin.updateArticle' : 'admin.publishArticle') }}</button><button type="submit" value="draft">{{ t(articlePublished ? 'admin.withdrawArticle' : 'admin.saveDraft') }}</button><a v-if="articlePublished" class="admin-public-link" :href="announcementUrl(articlesEditor.id)" target="_blank" rel="noopener noreferrer">{{ t('admin.viewArticle') }}</a></div>
          </fieldset>
        </form>
        <div class="admin-table-wrap articles-table"><table><thead><tr><th>{{ t('admin.articleTitle') }}</th><th>{{ t('admin.status') }}</th><th>{{ t('admin.date') }}</th><th>{{ t('admin.actions') }}</th></tr></thead><tbody><tr v-for="row in sortedArticles" :key="row.id"><td class="article-list-title"><span lang="zh-CN">{{ row.data.title }}</span><span v-if="row.data.pinned" class="article-pin">{{ t('admin.pinned') }}</span></td><td>{{ t(row.data.published ? 'admin.published' : 'admin.draft') }}</td><td><time :datetime="row.data.date">{{ d(new Date(row.data.date), 'article') }}</time></td><td><div class="row-actions"><button :disabled="busy" @click="editArticle(row)">{{ t('admin.edit') }}</button><a v-if="row.data.published" class="admin-public-link" :href="announcementUrl(row.id)" target="_blank" rel="noopener noreferrer">{{ t('admin.viewArticle') }}</a><button :disabled="busy" @click="deleteArticle(row)">{{ t('admin.remove') }}</button></div></td></tr></tbody></table><p v-if="!articles.length">{{ t('admin.empty') }}</p></div>
      </template>
      <form v-if="tab === 'settings' && settings" class="admin-form" @submit.prevent="run(async () => { settings = await saveRecord('settings', settings!) })">
        <label>{{ t('admin.contentLanguage') }}<select v-model="editLocale"><option v-for="language in supportedLocales" :key="language" :value="language">{{ t('common.languageName', {}, { locale: language }) }}</option></select></label>
        <label v-for="field in localizedFields" :key="field">{{ t(field === 'name' ? 'admin.siteName' : `admin.${field}`) }}<textarea v-model="settings.data[field][editLocale]" rows="2" /></label>
        <label>{{ t('admin.email') }}<input v-model="settings.data.email" type="email" /></label>
        <div class="admin-columns"><div v-for="field in imageFields" :key="field" class="image-field"><label>{{ t(`admin.${field}`) }}<input v-model="settings.data[field]" /></label><img :src="settings.data[field]" alt="" /><label class="upload-label">{{ t('admin.upload') }}<input type="file" accept="image/png,image/jpeg" :disabled="busy" @change="run(() => upload($event, field))" /></label></div></div>
        <section class="admin-related-links" aria-labelledby="related-links-settings">
          <h3 id="related-links-settings">{{ t('admin.relatedLinks') }}</h3>
          <div v-for="(link, index) in settings.data.relatedLinks" :key="index" class="admin-array" role="group" :aria-label="t('admin.linkEntry', { index: index + 1 })">
            <label>{{ t('admin.articleTitle') }}<input v-model="link.title[editLocale]" maxlength="200" /></label>
            <label>{{ t('admin.linkType') }}<select v-model="link.kind" @change="link.kind === 'help' && (link.url = '')"><option value="url">{{ t('admin.urlLink') }}</option><option value="help">{{ t('admin.helpLink') }}</option></select></label>
            <label v-if="link.kind === 'url'">{{ t('admin.url') }}<input v-model="link.url" required /></label>
            <div class="admin-actions"><button type="button" :disabled="index === 0" @click="moveRelatedLink(index, -1)">{{ t('admin.moveUp') }}</button><button type="button" :disabled="index === settings.data.relatedLinks.length - 1" @click="moveRelatedLink(index, 1)">{{ t('admin.moveDown') }}</button><button type="button" @click="settings.data.relatedLinks.splice(index, 1)">{{ t('admin.remove') }}</button></div>
          </div>
          <button type="button" @click="settings.data.relatedLinks.push({ title: { 'zh-CN': '', 'en-US': '' }, kind: 'url', url: '' })">{{ t('admin.add') }}</button>
        </section>
        <h3>{{ t('admin.links') }}</h3><div v-for="(link, index) in settings.data.links" :key="index" class="admin-array"><label>{{ t('admin.articleTitle') }}<input v-model="link.title[editLocale]" /></label><label>{{ t('admin.url') }}<input v-model="link.url" type="url" required /></label><button type="button" @click="settings.data.links.splice(index,1)">{{ t('admin.remove') }}</button></div><button type="button" @click="settings.data.links.push({ title: { 'zh-CN': '', 'en-US': '' }, url: '' })">{{ t('admin.add') }}</button>
        <h3>{{ t('admin.downloads') }}</h3><div v-for="(download,index) in settings.data.downloads" :key="index" class="admin-array"><label>{{ t('admin.name') }}<input v-model="download.name" required /></label><label>{{ t('admin.version') }}<input v-model="download.version" /></label><label>{{ t('admin.url') }}<input v-model="download.url" type="url" required /></label><button type="button" @click="settings.data.downloads.splice(index,1)">{{ t('admin.remove') }}</button></div><button type="button" @click="settings.data.downloads.push({ name: '', version: '', url: '' })">{{ t('admin.add') }}</button><button class="primary" :disabled="busy">{{ t('admin.save') }}</button>
      </form>
      <template v-if="tab === 'config' && config">
        <p>{{ t('admin.intervalHint') }}</p><form class="admin-form" @submit.prevent="run(previewConfig)"><label>{{ t('admin.config') }}<select v-model="configScope" @change="draft = null"><option value="global">{{ t('admin.global') }}</option><option value="mirror">{{ t('admin.mirrorConfig') }}</option></select></label>
          <div v-if="configScope === 'global'" class="admin-columns"><label>{{ t('admin.concurrent') }}<input v-model.number="config.global.concurrent" type="number" min="1" max="64" required /></label><label v-for="field in syncFields" :key="field">{{ t(`admin.${field}`) }}<input v-model.number="config.global[field]" type="number" :min="field === 'retry' ? 0 : field === 'timeout' ? 30 : 1" :max="field === 'retry' ? 20 : field === 'timeout' ? 1209600 : 525600" required /></label></div>
          <template v-else><label>{{ t('admin.mirrors') }}<select v-model="selectedConfig" @change="chooseMirrorConfig"><option value="" disabled>{{ t('admin.mirrors') }}</option><option v-for="row in config.mirrors" :key="row.name" :value="row.name">{{ row.name }}</option><option v-if="config.canAdd" value="__new">{{ t('admin.newMirror') }}</option></select></label><template v-if="mirrorConfig"><label>{{ t('admin.name') }}<input v-model="mirrorConfig.name" :readonly="selectedConfig !== '__new'" pattern="[a-zA-Z0-9][a-zA-Z0-9_.+\-]*" required /></label><label>{{ t('admin.provider') }}<select v-model="mirrorConfig.provider" :disabled="selectedConfig !== '__new'"><option value="rsync">rsync</option><option value="two-stage-rsync">two-stage-rsync</option><option v-if="!['rsync','two-stage-rsync'].includes(mirrorConfig.provider)" :value="mirrorConfig.provider">{{ mirrorConfig.provider }}</option></select></label><label>{{ t('admin.upstream') }}<input v-model="mirrorConfig.upstream" required /></label><div class="admin-columns"><label v-for="field in syncFields" :key="field">{{ t(`admin.${field}`) }}<input v-model.number="mirrorConfig[field]" type="number" :min="field === 'retry' ? 0 : field === 'timeout' ? 30 : 1" :max="field === 'retry' ? 20 : field === 'timeout' ? 1209600 : 525600" required /></label></div></template></template>
          <button class="primary" :disabled="busy || (configScope === 'mirror' && !mirrorConfig)">{{ t('admin.previewConfig') }}</button>
        </form>
        <section v-if="draft" class="admin-editor"><h3>{{ t('admin.preview') }}</h3><div v-for="(file,path) in draft.files" :key="path"><h4 class="file-name">{{ path }}</h4><div class="admin-columns diff"><div><h4>{{ t('admin.before') }}</h4><pre>{{ file.before }}</pre></div><div><h4>{{ t('admin.after') }}</h4><pre>{{ file.after }}</pre></div></div></div><button class="primary" :disabled="busy" @click="applyConfig">{{ t('admin.apply') }}</button></section>
        <h3>{{ t('admin.versions') }}</h3><div class="admin-table-wrap"><table><thead><tr><th>{{ t('admin.time') }}</th><th>{{ t('admin.actor') }}</th><th>{{ t('admin.status') }}</th><th>{{ t('admin.actions') }}</th></tr></thead><tbody><tr v-for="version in versions" :key="version.id"><td>{{ date(version.created) }}</td><td>{{ version.username }}</td><td>{{ eventLabel(version.status) }}</td><td><button v-if="version.status === 'applied'" :disabled="busy" @click="rollback(version.id)">{{ t('admin.rollback') }}</button></td></tr></tbody></table></div>
      </template>
      <template v-if="tab === 'users'">
        <h3>{{ t('admin.changePassword') }}</h3><p>{{ t('admin.passwordHint') }}</p><form class="admin-form admin-login" @submit.prevent="run(changePassword)"><label>{{ t('admin.currentPassword') }}<input v-model="passwords.currentPassword" type="password" autocomplete="current-password" required /></label><label>{{ t('admin.newPassword') }}<input v-model="passwords.newPassword" type="password" autocomplete="new-password" minlength="12" maxlength="256" required /></label><label>{{ t('admin.confirmPassword') }}<input v-model="passwords.confirm" type="password" autocomplete="new-password" minlength="12" maxlength="256" required /></label><button class="primary" :disabled="busy">{{ t('admin.changePassword') }}</button></form>
        <template v-if="session.role === 'admin'"><h3>{{ t('admin.users') }}</h3><div class="admin-table-wrap"><table><thead><tr><th>{{ t('admin.username') }}</th><th>{{ t('admin.role') }}</th></tr></thead><tbody><tr v-for="user in users" :key="user.id"><td>{{ user.username }}</td><td>{{ t(`admin.${user.role}`) }}</td></tr></tbody></table></div><h3>{{ t('admin.newUser') }}</h3><form class="admin-form admin-login" @submit.prevent="run(createUser)"><label>{{ t('admin.username') }}<input v-model="newUser.username" autocomplete="off" pattern="[a-zA-Z0-9][a-zA-Z0-9_.+\-]*" required /></label><label>{{ t('admin.password') }}<input v-model="newUser.password" type="password" autocomplete="new-password" minlength="12" maxlength="256" required /></label><label>{{ t('admin.role') }}<select v-model="newUser.role"><option value="admin">{{ t('admin.admin') }}</option><option value="editor">{{ t('admin.editor') }}</option></select></label><button class="primary" :disabled="busy">{{ t('admin.newUser') }}</button></form></template>
      </template>
      <template v-if="tab === 'audit'">
        <div class="admin-table-wrap"><table id="admin-audit-table"><thead><tr><th>{{ t('admin.time') }}</th><th>{{ t('admin.actor') }}</th><th>{{ t('admin.action') }}</th><th>{{ t('admin.target') }}</th><th>{{ t('admin.result') }}</th></tr></thead><tbody><tr v-for="entry in audit.items" :key="entry.id"><td>{{ date(entry.created) }}</td><td>{{ entry.username }}</td><td>{{ eventLabel(entry.action) }}</td><td class="file-name">{{ entry.target }}</td><td>{{ eventLabel(entry.result) }}</td></tr></tbody></table><p v-if="!audit.items.length">{{ t('admin.empty') }}</p></div>
        <nav v-if="audit.totalPages > 1" class="admin-pagination" :aria-label="t('admin.auditPagination')">
          <span>{{ t('admin.auditTotal', { total: n(audit.total) }) }}</span>
          <div class="admin-page-controls">
            <button :disabled="busy || audit.page <= 1" aria-controls="admin-audit-table" @click="run(() => loadAudit(audit.page - 1))">{{ t('admin.previousPage') }}</button>
            <span role="status" aria-live="polite">{{ t('admin.pagePosition', { page: n(audit.page), pages: n(audit.totalPages) }) }}</span>
            <button :disabled="busy || audit.page >= audit.totalPages" aria-controls="admin-audit-table" @click="run(() => loadAudit(audit.page + 1))">{{ t('admin.nextPage') }}</button>
          </div>
        </nav>
      </template>
    </template>
    <dialog ref="confirmDialog" class="admin-confirm" :aria-label="t('admin.confirm')" @close="confirmation = null"><p>{{ confirmation?.message }}</p><div class="admin-actions"><button @click="confirmDialog?.close()">{{ t('admin.cancel') }}</button><button class="primary" @click="confirmed">{{ t('admin.confirm') }}</button></div></dialog>
  </section>
</template>

<style scoped>
.admin-page { min-width: 0; padding: 12px 0 40px; }
.admin-top, .admin-toolbar { display: flex; justify-content: space-between; align-items: center; gap: 16px; flex-wrap: wrap; margin-bottom: 22px; }
h1 { margin: 0; font-size: 32px; } h2 { margin: 0; font-size: 25px; } h3 { margin: 24px 0 14px; font-size: 20px; } h4 { margin: 14px 0 8px; }
.admin-actions, .row-actions { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
button { min-height: 38px; padding: 7px 13px; border: 1px solid var(--line); border-radius: 4px; color: var(--ink); background: var(--panel); cursor: pointer; font-size: 14px; }
button:hover { border-color: var(--blue); color: var(--blue); } button:disabled { opacity: .5; cursor: wait; } .primary { background: var(--blue); color: white; border-color: var(--blue); } .primary:hover { color: white; filter: brightness(1.1); }
.admin-tabs { display: flex; gap: 6px; overflow-x: auto; padding: 4px 0 18px; margin-bottom: 22px; }
.admin-tabs button { flex-shrink: 0; border-color: transparent; background: transparent; } .admin-tabs [aria-current] { border-bottom: 2px solid var(--blue); color: var(--blue); border-radius: 0; font-weight: 700; }
.admin-error, .admin-notice { padding: 12px 16px; border-left: 3px solid var(--blue); background: var(--panel); overflow-wrap: anywhere; } .admin-error { border-color: #d65252; }
.admin-form { display: grid; gap: 18px; max-width: 1100px; } .admin-login { max-width: 420px; }
label { display: flex; flex-direction: column; gap: 8px; min-width: 0; font-size: 14px; font-weight: 500; }
input, textarea, select { width: 100%; min-width: 0; min-height: 40px; padding: 9px 11px; border: 1px solid var(--line); border-radius: 4px; background: var(--panel); color: var(--ink); font: inherit; }
textarea { resize: vertical; line-height: 1.7; } .check { flex-direction: row; align-items: center; gap: 8px; } .check input { width: 18px; min-height: 18px; accent-color: var(--blue); }
.admin-form > button { justify-self: start; } .admin-columns { display: grid; grid-template-columns: repeat(2,minmax(0,1fr)); gap: 18px; } .admin-array { display: flex; flex-wrap: wrap; gap: 12px; align-items: end; padding: 14px 0; border-bottom: 1px solid var(--line); } .admin-array label { flex: 1; min-width: 160px; }
.admin-related-links > button { margin-top: 16px; } .admin-related-links .admin-actions { gap: 6px; }
.admin-public-link { color: var(--blue); font-size: 14px; text-underline-offset: 3px; } .admin-public-link:hover { text-decoration: underline; }
.announcement-editor { max-width: none; scroll-margin-top: 20px; }
.announcement-editor .admin-toolbar { margin-bottom: 0; } .announcement-editor h3 { margin: 0; }
.announcement-editor fieldset { display: grid; gap: 22px; min-width: 0; padding: 0; margin: 0; border: 0; }
.article-options { display: flex; gap: 24px; align-items: center; flex-wrap: wrap; } .article-options > label:first-child { width: 200px; } .article-options .check { padding-top: 24px; }
.article-state { margin-left: auto; color: var(--muted); font-size: 14px; }
.article-workspace { display: grid; grid-template-columns: repeat(2, minmax(0,1fr)); gap: 28px; align-items: start; }
.article-source textarea { min-height: 380px; font-family: ui-monospace, Consolas, monospace; font-size: 14px; }
.article-preview { min-width: 0; overflow-wrap: anywhere; } .article-preview h4 { margin: 0 0 12px; font-size: 14px; font-weight: 500; }
.article-preview h2 { font-size: 24px; color: var(--blue); line-height: 1.5; } .article-preview time { display: block; margin: 10px 0 22px; color: var(--muted); font-size: 14px; }
.article-preview :deep(.markdown-content) { font-size: 16px; line-height: 1.9; } .article-empty { color: var(--muted); font-size: 14px; }
.article-view-switch { display: none; gap: 8px; } .article-view-switch [aria-pressed='true'] { color: var(--blue); border-color: var(--blue); }
.article-list-title { overflow-wrap: anywhere; } .article-pin { margin-left: 10px; color: var(--blue); font-size: 12px; }
.articles-table td:not(:first-child) { white-space: nowrap; }
@media (max-width: 900px) {
  .article-workspace { display: block; } .article-view-switch { display: flex; }
  .article-workspace[data-view='write'] .article-preview, .article-workspace[data-view='preview'] .article-source { display: none; }
  .article-preview > h4 { display: none; }
}
@media (max-width: 640px) {
  .article-options { gap: 16px; } .article-options > label:first-child { width: min(200px, 100%); } .article-options .check { padding-top: 0; }
  .article-submit button { min-height: 44px; }
  .articles-table table, .articles-table tbody { display: block; } .articles-table thead { display: none; }
  .articles-table tr { display: grid; grid-template-columns: auto 1fr; gap: 10px 16px; padding: 18px 0; border-bottom: 1px solid var(--line); }
  .articles-table td { padding: 0; border: 0; } .articles-table td:first-child, .articles-table td:last-child { grid-column: 1 / -1; }
}
.admin-table-wrap { max-width: 100%; overflow: auto; margin: 20px 0; } table { width: 100%; border-collapse: collapse; font-size: 14px; } th, td { padding: 12px 10px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: middle; } thead th { font-weight: 700; white-space: nowrap; } tbody th { font-weight: 500; }
.admin-pagination, .admin-page-controls { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; }
.admin-pagination { justify-content: space-between; font-size: 14px; }
.admin-page-controls { gap: 10px; font-variant-numeric: tabular-nums; }
.admin-page-controls > span { text-align: center; }
.admin-page-controls button:disabled { cursor: default; }
tr[data-status='syncing'] { background: color-mix(in srgb, var(--blue) 16%,transparent); } tr[data-status='failed'] { background: #df4d4d1c; } .row-actions { gap: 6px; min-width: 260px; } .row-actions button { font-size: 12px; padding: 4px 8px; min-height: 32px; }
.admin-summary { display: flex; flex-wrap: wrap; gap: 28px 50px; } dt { color: var(--muted); font-size: 14px; } dd { margin: 10px 0 0; font-size: 24px; overflow-wrap: anywhere; }
.admin-editor { margin: 30px 0; padding: 24px; border: 1px solid var(--line); border-radius: 5px; } .admin-editor h3 { margin-top: 0; } .admin-search { max-width: 360px; }
.admin-log, .diff pre { padding: 16px; background: var(--panel); white-space: pre-wrap; overflow-wrap: anywhere; font: 12px/1.7 ui-monospace,monospace; max-height: 520px; overflow: auto; } .file-name { overflow-wrap: anywhere; }
.image-field { display: grid; gap: 10px; align-content: start; } .image-field img { max-width: 230px; height: 65px; object-fit: contain; object-position: left; } .upload-label { font-size: 12px; } .upload-label input { font-size: 12px; }
.admin-confirm { width: min(480px,calc(100% - 32px)); padding: 26px; border: 1px solid var(--line); border-radius: 6px; color: var(--ink); background: var(--panel); } .admin-confirm::backdrop { background: #0008; } .admin-confirm p { margin: 0 0 24px; line-height: 1.8; }
@media(max-width:700px) {
 .admin-pagination { align-items: start; gap: 12px; }
 .admin-page-controls { width: 100%; justify-content: space-between; gap: 6px; }
 .admin-page-controls button { min-height: 42px; padding: 7px 10px; }
 .jobs-table table, .jobs-table tbody { display: block; }
 .jobs-table thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); }
 .jobs-table tr { display: grid; grid-template-columns: minmax(0,1fr) auto auto; gap: 12px; padding: 16px 10px; border-bottom: 1px solid var(--line); }
 .jobs-table :is(td,th) { padding: 0; border: 0; overflow-wrap: anywhere; }
 .jobs-table td:last-child { grid-column: 1/-1; }
 .jobs-table .row-actions { min-width: 0; }
 .jobs-table .row-actions button { min-height: 40px; }
 h1 { font-size: 28px; } .admin-columns { grid-template-columns: 1fr; } .admin-editor { padding: 16px; } .admin-tabs { margin-right: -4px; } th, td { padding: 10px 8px; } .admin-summary { gap: 24px; } dd { font-size: 20px; } .admin-top { align-items: start; } }
</style>
