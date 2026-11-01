import { mkdir, readFile, writeFile, unlink } from 'node:fs/promises'
import { createI18n } from 'vue-i18n'
import { directoryMessages } from '../src/directory/messages.ts'

const i18n = createI18n({ legacy: false, locale: 'zh-CN', messages: directoryMessages })
const t = (key: string) => i18n.global.t(key)
const escape = (value: string) => value.replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[char]!)
const text = (key: string) => `<span data-i18n="${key}">${escape(t(key))}</span>`
const navigation = [
  ['mirrors', '/'], ['news', '/announcements'], ['status', '/status'],
  ['feedback', 'https://git.itouc.cn/ITStudio_OUC/OUCMirror-WebUI/issues'],
].map(([key, href]) => `<a class="nav-link${key === 'mirrors' ? ' active' : ''}" href="${href}">${text(`common.nav.${key}`)}</a>`).join('')
const languages = Object.entries(directoryMessages).map(([locale, messages]) => `<button class="language-option" data-locale="${locale}" lang="${locale}">${escape(messages.common.languageName)}</button>`).join('')
const themes = ['light', 'dark', 'system'].map(theme => `<button data-theme-choice="${theme}">${text(`common.themes.${theme}`)}</button>`).join('')
const specials: Record<string, string> = { navigation, languages, themes }
const html = (await readFile('dist/directory.html', 'utf8')).replace(/\{\{([\w.]+)\}\}/g, (_, key) => {
  if (key in specials) return specials[key]!
  if (!i18n.global.te(key)) throw new Error(`Missing directory template message: ${key}`)
  return escape(t(key))
})
const parts = html.split('<!-- DIRECTORY_LIST -->')
if (parts.length !== 2) throw new Error('Expected exactly one Nginx directory insertion point')
await mkdir('dist/fancy-index', { recursive: true })
await writeFile('dist/fancy-index/header.html', parts[0]!)
await writeFile('dist/fancy-index/footer.html', parts[1]!)
// Translate the four native Fancyindex labels on the server as well, so the
// default Chinese listing is complete and accessible even without JavaScript.
const replacements = [
  ['File Name</a>', `${text('directory.name')}</a>`],
  ['File Size</a>', `${text('directory.size')}</a>`],
  ['Date</a>', `${text('directory.modified')}</a>`],
  ['Parent directory/</a>', `${text('directory.parent')}</a>`],
]
const quote = (value: string) => '"' + value.replace(/\\/g, '\\\\').replace(/\$/g, '\\$').replace(/"/g, '\\"') + '"'
await writeFile('dist/fancy-index/labels.conf', 'sub_filter_once on;\n' + replacements.map(([before, after]) => `sub_filter ${quote(before!)} ${quote(after!)};`).join('\n') + '\n')
// Only the two internal fragments are served; Nginx inserts the table between them.
await unlink('dist/directory.html')
console.log('Built Nginx directory header, footer and locale labels.')
