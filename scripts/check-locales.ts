import commonZh from '../src/locales/zh-CN/common.ts'
import commonEn from '../src/locales/en-US/common.ts'
import mirrorsZh from '../src/locales/zh-CN/mirrors.ts'
import mirrorsEn from '../src/locales/en-US/mirrors.ts'
import announcementsZh from '../src/locales/zh-CN/announcements.ts'
import announcementsEn from '../src/locales/en-US/announcements.ts'
import monitorZh from '../src/locales/zh-CN/monitor.ts'
import monitorEn from '../src/locales/en-US/monitor.ts'
import { datetimeFormats as datetimeZh, numberFormats as numbersZh } from '../src/locales/zh-CN/formats.ts'
import { datetimeFormats as datetimeEn, numberFormats as numbersEn } from '../src/locales/en-US/formats.ts'
import { createI18n } from 'vue-i18n'
import { adminZh, adminEn } from '../src/locales/admin-messages.ts'
import directoryZh from '../src/locales/zh-CN/directory.ts'
import directoryEn from '../src/locales/en-US/directory.ts'
import errorsZh from '../src/locales/zh-CN/errors.ts'
import errorsEn from '../src/locales/en-US/errors.ts'

function flatten(value: Record<string, unknown>, prefix = ''): Record<string, string> {
  return Object.fromEntries(Object.entries(value).flatMap(([key, entry]) => {
    const path = prefix ? `${prefix}.${key}` : key
    return typeof entry === 'string' ? [[path, entry]] : Object.entries(flatten(entry as Record<string, unknown>, path))
  }))
}

const zh = flatten({ common: commonZh, mirrors: mirrorsZh, announcements: announcementsZh, monitor: monitorZh, admin: adminZh, directory: directoryZh, errors: errorsZh })
const en = flatten({ common: commonEn, mirrors: mirrorsEn, announcements: announcementsEn, monitor: monitorEn, admin: adminEn, directory: directoryEn, errors: errorsEn })
const errors: string[] = []
const formats = {
  'zh-CN': { datetimeFormats: datetimeZh, numberFormats: numbersZh },
  'en-US': { datetimeFormats: datetimeEn, numberFormats: numbersEn },
}
for (const kind of ['datetimeFormats', 'numberFormats'] as const) {
  const zhNames = Object.keys(formats['zh-CN'][kind])
  const enNames = Object.keys(formats['en-US'][kind])
  for (const name of new Set([...zhNames, ...enNames])) {
    if (!zhNames.includes(name) || !enNames.includes(name)) errors.push(`Missing format: ${kind}.${name}`)
  }
}
for (const key of new Set([...Object.keys(zh), ...Object.keys(en)])) {
  if (!(key in zh) || !(key in en)) { errors.push(`Missing key: ${key}`); continue }
  const params = (value: string) => [...new Set([...value.matchAll(/\{(\w+)\}/g)].map(match => match[1]))].sort().join(',')
  if (params(zh[key]!) !== params(en[key]!)) errors.push(`Mismatched parameters: ${key}`)
}
if (errors.length) throw new Error(errors.join('\n'))
// Compile every message too: special characters such as @ can be valid TS but invalid i18n syntax.
for (const [locale, messages] of Object.entries({ 'zh-CN': zh, 'en-US': en })) {
  const definitions = formats[locale as keyof typeof formats]
  const i18n = createI18n({
    legacy: false, locale, flatJson: true, messages: { [locale]: { ...messages } },
    datetimeFormats: { [locale]: definitions.datetimeFormats },
    numberFormats: { [locale]: definitions.numberFormats },
  })
  for (const [key, value] of Object.entries(messages)) {
    const params = Object.fromEntries([...value.matchAll(/\{(\w+)\}/g)].map(match => [match[1], '1']))
    try { i18n.global.t(key, params) }
    catch (error) { errors.push(`${locale}.${key}: ${String(error)}`) }
  }
  for (const name of Object.keys(definitions.datetimeFormats)) {
    try { i18n.global.d(new Date('2026-11-01T00:00:00+08:00'), name) }
    catch (error) { errors.push(`${locale}.datetimeFormats.${name}: ${String(error)}`) }
  }
  for (const name of Object.keys(definitions.numberFormats)) {
    try { i18n.global.n(1234.5, name) }
    catch (error) { errors.push(`${locale}.numberFormats.${name}: ${String(error)}`) }
  }
}
if (errors.length) throw new Error(errors.join('\n'))
console.log(`Locale check passed: ${Object.keys(zh).length} keys, interpolation parameters, message syntax and format definitions match.`)
