import { inject, onBeforeUnmount, onMounted, provide, ref, shallowRef, type InjectionKey, type Ref, type ShallowRef } from 'vue'
import { api } from './api'
import { defaultGeneratorCatalog, parseGeneratorCatalog, type GeneratorCatalog } from './generator-catalog'

interface CatalogContext {
  catalog: ShallowRef<GeneratorCatalog | null>
  failed: Ref<boolean>
  refresh: () => Promise<void>
}
const catalogKey: InjectionKey<CatalogContext> = Symbol('generator-catalog')

export function provideGeneratorCatalog(enabled: boolean) {
  const catalog = shallowRef<GeneratorCatalog | null>(null)
  const failed = ref(false)
  let controller: AbortController | undefined
  let timer: ReturnType<typeof setInterval> | undefined
  let disposed = false
  async function refresh() {
    controller?.abort()
    const request = new AbortController()
    controller = request
    try {
      const result = parseGeneratorCatalog(await api('/generators', { signal: request.signal }))
      if (!result) throw new Error('Invalid generator catalog')
      if (disposed || request.signal.aborted) return
      if (JSON.stringify(catalog.value) !== JSON.stringify(result)) catalog.value = result
      failed.value = false
    } catch {
      if (!disposed && !request.signal.aborted) failed.value = true
    }
  }
  provide(catalogKey, { catalog, failed, refresh })
  onMounted(() => { if (enabled) { void refresh(); timer = setInterval(() => { if (!document.hidden) void refresh() }, 15000) } })
  onBeforeUnmount(() => { disposed = true; controller?.abort(); clearInterval(timer) })
}

export function useGeneratorCatalog(): CatalogContext {
  // Isolated previews can use the shared seed; the application always provides
  // backend data and does not silently substitute defaults after a fetch failure.
  return inject(catalogKey, { catalog: shallowRef(defaultGeneratorCatalog), failed: ref(false), refresh: async () => {} })
}
