import { onBeforeUnmount, onMounted, ref } from 'vue'
import { api, ApiError, type AnnouncementContent, type ContentRecord, type SiteSettings } from './api'
import type { Mirror } from './data'

interface PublicPayload {
  site: SiteSettings
  mirrors: Omit<Mirror, 'mark' | 'color' | 'descriptionKey'>[]
  announcements: ContentRecord<AnnouncementContent>[]
  source: { updatedAt: number; stale: boolean; code: string }
}
export function usePublicData(enabled = true) {
  const site = ref<SiteSettings | null>(null)
  const mirrors = ref<Mirror[]>([])
  const announcements = ref<ContentRecord<AnnouncementContent>[]>([])
  const source = ref({ updatedAt: 0, stale: false, code: '' })
  const loading = ref(enabled)
  let controller: AbortController | undefined
  let timer: ReturnType<typeof setInterval> | undefined
  let disposed = false
  async function refresh() {
    if (!enabled || disposed) return
    controller?.abort(); controller = new AbortController()
    try {
      const data = await api<PublicPayload>('/public', { signal: controller.signal })
      if (disposed) return
      site.value = data.site
      mirrors.value = data.mirrors.map(mirror => ({ ...mirror, mark: mirror.name[0]?.toUpperCase() || '', color: '#004098', descriptionKey: '' }))
      announcements.value = data.announcements
      source.value = data.source
    } catch (error) {
      if ((error as Error).name !== 'AbortError') source.value = { ...source.value, stale: true, code: error instanceof ApiError ? error.code : 'NETWORK_ERROR' }
    } finally { if (!disposed) loading.value = false }
  }
  onMounted(() => { if (!enabled) return; void refresh(); timer = setInterval(() => { if (!document.hidden) void refresh() }, 15000) })
  onBeforeUnmount(() => { disposed = true; controller?.abort(); clearInterval(timer) })
  return { site, mirrors, announcements, source, loading, refresh }
}
