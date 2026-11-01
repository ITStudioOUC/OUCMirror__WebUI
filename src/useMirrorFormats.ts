import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Mirror } from './data'

export function useMirrorFormats(now: Readonly<Ref<number>>) {
  const { t, d, n } = useI18n({ useScope: 'global' })
  function mirrorSize(mirror: Mirror) {
    if (mirror.size === null) return t('common.emptyValue')
    const unit = mirror.size >= 1e12 ? 'terabyte' : 'gigabyte'
    return t(`mirrors.size.${unit}`, { value: n(mirror.size / (unit === 'terabyte' ? 1e12 : 1e9), 'size') })
  }
  function relativeTime(timestamp: number | null, next = false) {
    if (timestamp === null) return next ? t('mirrors.unavailable') : t('common.emptyValue')
    const diff = (timestamp - now.value) / 60_000
    if (next && diff <= 1) return t('mirrors.soon')
    if (Math.abs(diff) < 1) return t('mirrors.justNow')
    const unit = Math.abs(diff) >= 1440 ? 'day' : Math.abs(diff) >= 60 ? 'hour' : 'minute'
    const value = Math.round(diff / (unit === 'day' ? 1440 : unit === 'hour' ? 60 : 1))
    const count = Math.abs(value)
    return t(`mirrors.relativeTime.${value < 0 ? 'past' : 'future'}.${unit}`, { count: n(count, 'integer') }, count)
  }
  function tableNextTime(mirror: Mirror) {
    if (mirror.nextSync === null) return t('common.emptyValue')
    return mirror.nextSync - now.value <= 60_000 ? t('mirrors.tableSoon') : relativeTime(mirror.nextSync, true)
  }
  function fullTime(timestamp: number | null) {
    return timestamp === null ? undefined : d(timestamp, 'timestamp')
  }
  return { mirrorSize, relativeTime, tableNextTime, fullTime }
}
