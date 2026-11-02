import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { errorCodes, errorMessages, apiErrorCodes } from '../src/error-pages.ts'
import { initializeErrorPage } from '../src/error-page.js'

const escape = (value: string) => value.replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[char]!)
const runtime = `(${initializeErrorPage.toString()})(${JSON.stringify(errorMessages)});\n`
const script = `/assets/error-pages-${createHash('sha256').update(runtime).digest('hex').slice(0, 12)}.js`
const template = await readFile('error.html', 'utf8')
const messages = errorMessages['zh-CN']
const languages = Object.entries(errorMessages).map(([locale, value]) => `<option value="${locale}" lang="${locale}">${escape(value.common.languageName)}</option>`).join('')
const themes = Object.entries(messages.common.themes).map(([value, label]) => `<option value="${value}">${escape(label)}</option>`).join('')
await mkdir('dist/errors', { recursive: true })
await writeFile('dist' + script, runtime)
for (const status of errorCodes) {
  const state = messages.errors.codes[status as keyof typeof messages.errors.codes]
  const values: Record<string, string> = { status, script, title: state.title, description: state.description, pageTitle: `${status} · ${state.title} · ${messages.common.brand}`, siteName: messages.common.siteName, language: messages.common.language, theme: messages.common.theme, home: messages.errors.home, retry: messages.errors.retry }
  const html = template.replace(/\{\{(\w+)\}\}/g, (_, key) => key === 'languages' ? languages : key === 'themes' ? themes : escape(values[key]!))
  await writeFile(`dist/errors/${status}.html`, html)
  await writeFile(`dist/errors/api-${status}.json`, JSON.stringify({ code: apiErrorCodes[status] }) + '\n')
}
console.log(`Built ${errorCodes.length} independent error pages and JSON gateway responses.`)
