<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy } from 'lucide-vue-next'
import { copyText } from '../copy-text'

const props = defineProps<{ code: string; caption: string }>()
const { t } = useI18n()
const copied = ref(false)
const failed = ref(false)
let revision = 0
let disposed = false
let reset: ReturnType<typeof setTimeout> | undefined
function resetCopy() { revision++; copied.value = false; failed.value = false; clearTimeout(reset) }
watch(() => props.code, resetCopy)
async function copy() {
  const current = ++revision
  clearTimeout(reset)
  try {
    await copyText(props.code)
    if (disposed || current !== revision) return
    copied.value = true; failed.value = false
  } catch {
    if (disposed || current !== revision) return
    copied.value = false; failed.value = true
  }
  reset = setTimeout(resetCopy, 2500)
}
onBeforeUnmount(() => { disposed = true; clearTimeout(reset) })
</script>

<template>
  <div class="source-output">
    <div class="source-code">
      <div class="source-code-heading"><span>{{ caption }}</span><button type="button" :aria-label="t(copied ? 'common.copied' : 'common.copy')" :title="t(copied ? 'common.copied' : 'common.copy')" @click="copy"><Check v-if="copied" :size="18" aria-hidden="true" /><Copy v-else :size="18" aria-hidden="true" /></button></div>
      <pre tabindex="0" :aria-label="t('mirrors.generator.generated')"><code>{{ code }}</code></pre>
    </div>
    <p class="copy-status" :class="{ 'sr-only': !failed }" role="status">{{ failed ? t('common.copyFailed') : copied ? t('common.copied') : '' }}</p>
  </div>
</template>

<style scoped>
.source-output + .source-output { margin-top: 16px; }
.source-code { min-width: 0; overflow: hidden; border: 1px solid var(--line); border-radius: 4px; background: var(--surface); }
.source-code-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 7px 10px 0 14px; color: var(--ink-2); font: 12px/1.5 ui-monospace, Consolas, monospace; }
.source-code-heading > span { min-width: 0; overflow-wrap: anywhere; }
.source-code-heading button { display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 3px; color: var(--ink); }
.source-code-heading button:hover { color: var(--blue); background: var(--blue-soft); }
.source-code pre { max-width: 100%; overflow: auto; margin: 0; padding: 6px 14px 16px; color: var(--ink); font: 13px/1.85 ui-monospace, 'Cascadia Code', Consolas, monospace; white-space: pre; }
.copy-status { margin: 10px 0; color: var(--ink); font-size: 14px; line-height: 1.7; }
@media (max-width: 640px) {
  .source-code-heading button { width: 44px; height: 44px; }
  .source-code pre { font-size: 12px; }
}
</style>
