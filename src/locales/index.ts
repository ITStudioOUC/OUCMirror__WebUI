import { createI18n } from 'vue-i18n'
import commonZh from './zh-CN/common'
import mirrorsZh from './zh-CN/mirrors'
import announcementsZh from './zh-CN/announcements'
import commonEn from './en-US/common'
import mirrorsEn from './en-US/mirrors'
import announcementsEn from './en-US/announcements'
import monitorZh from './zh-CN/monitor'
import monitorEn from './en-US/monitor'
import { datetimeFormats as datetimeZh, numberFormats as numbersZh } from './zh-CN/formats'
import { datetimeFormats as datetimeEn, numberFormats as numbersEn } from './en-US/formats'

export const messages = {
  'zh-CN': { common: commonZh, mirrors: mirrorsZh, announcements: announcementsZh, monitor: monitorZh },
  'en-US': { common: commonEn, mirrors: mirrorsEn, announcements: announcementsEn, monitor: monitorEn },
}
export type Locale = keyof typeof messages
export const supportedLocales = Object.keys(messages) as Locale[]
export const datetimeFormats = { 'zh-CN': datetimeZh, 'en-US': datetimeEn }
export const numberFormats = { 'zh-CN': numbersZh, 'en-US': numbersEn }

export function readPreference(key: string) {
  try { return localStorage.getItem(key) } catch { return null }
}

export function savePreference(key: string, value: string) {
  try { localStorage.setItem(key, value) } catch { /* Preferences are optional. */ }
}

function initialLocale(): Locale {
  const saved = readPreference('mirrorouc.locale')
  if (saved && supportedLocales.includes(saved as Locale)) return saved as Locale
  for (const language of navigator.languages) {
    const preferred = language.toLowerCase()
    const match = supportedLocales.find(value => value.toLowerCase() === preferred)
      ?? supportedLocales.find(value => value.toLowerCase().split('-')[0] === preferred.split('-')[0])
    if (match) return match
  }
  return 'zh-CN'
}

export const i18n = createI18n<[typeof messages['zh-CN']], Locale>({
  legacy: false,
  locale: initialLocale(),
  fallbackLocale: 'zh-CN',
  messages,
  datetimeFormats,
  numberFormats,
})
