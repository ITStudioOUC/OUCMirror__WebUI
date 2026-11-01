<script setup lang="ts">
import { computed, onBeforeUnmount, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy } from 'lucide-vue-next'
import { debianReleases, debianSources, type DebianFormat, type DebianRelease } from '../debian-sources'
import { copyText } from '../copy-text'

const props = defineProps<{ format: DebianFormat }>()
const { t } = useI18n()
const id = useId()
const release = ref<DebianRelease>('trixie')
const sources = ref(false)
const nonFree = ref(true)
const securityMirror = ref(false)
const selection = computed(() => debianReleases.find(entry => entry.id === release.value)!)
const code = computed(() => debianSources({ origin: window.location.origin, release: release.value, format: props.format, sources: sources.value, nonFree: nonFree.value, securityMirror: securityMirror.value }))
const file = computed(() => props.format === 'list' ? '/etc/apt/sources.list' : '/etc/apt/sources.list.d/debian.sources')
const copied = ref(false)
const copyFailed = ref(false)
let reset: ReturnType<typeof setTimeout> | undefined
let revision = 0
let disposed = false
function resetCopy() { revision++; copied.value = false; copyFailed.value = false; clearTimeout(reset) }
watch(code, resetCopy)
watch(selection, value => { if (!value.stable) securityMirror.value = false })
async function copy() {
  const current = ++revision
  clearTimeout(reset)
  try {
    await copyText(code.value)
    if (disposed || current !== revision) return
    copied.value = true
    copyFailed.value = false
  } catch {
    if (disposed || current !== revision) return
    copied.value = false
    copyFailed.value = true
  }
  reset = setTimeout(resetCopy, 2500)
}
onBeforeUnmount(() => { disposed = true; clearTimeout(reset) })
</script>

<template>
  <section class="debian-sources" :data-format="format" :aria-label="t('mirrors.debian.generator', { format: format === 'list' ? t('mirrors.debian.traditional') : 'DEB822' })">
    <div class="source-controls">
      <label class="source-control source-version"><span>{{ t('mirrors.debian.version') }}</span><select v-model="release"><option v-for="entry in debianReleases" :key="entry.id" :value="entry.id">{{ entry.version ? t('mirrors.debian.release', { version: entry.version, codename: entry.id }) : entry.id }}</option></select></label>
      <label class="source-control"><span>{{ t('mirrors.debian.sources') }}</span><input v-model="sources" type="checkbox" role="switch" /><span class="source-toggle" aria-hidden="true" /></label>
      <label class="source-control"><span>{{ t('mirrors.debian.nonFree') }}</span><input v-model="nonFree" type="checkbox" role="switch" /><span class="source-toggle" aria-hidden="true" /></label>
      <label class="source-control" :class="{ 'source-disabled': !selection.stable }"><span>{{ t('mirrors.debian.securityMirror') }}</span><input v-model="securityMirror" type="checkbox" role="switch" :disabled="!selection.stable" :aria-describedby="!selection.stable ? `${id}-branch` : undefined" /><span class="source-toggle" aria-hidden="true" /></label>
    </div>
    <p v-if="!selection.stable" :id="`${id}-branch`" class="source-note">{{ t('mirrors.debian.rollingNotice') }}</p>
    <div class="source-code">
      <div class="source-code-heading"><span>{{ file }}</span><button type="button" :aria-label="t(copied ? 'common.copied' : 'common.copy')" :title="t(copied ? 'common.copied' : 'common.copy')" @click="copy"><Check v-if="copied" :size="18" aria-hidden="true" /><Copy v-else :size="18" aria-hidden="true" /></button></div>
      <pre tabindex="0" :aria-label="t('mirrors.debian.generated')"><code>{{ code }}</code></pre>
    </div>
    <p class="copy-status" :class="{ 'sr-only': !copyFailed }" role="status">{{ copyFailed ? t('common.copyFailed') : copied ? t('common.copied') : '' }}</p>
  </section>
</template>

<style scoped>
.debian-sources { min-width: 0; margin: 20px 0 30px; }
.source-controls { display: grid; grid-template-columns: minmax(180px, 1.35fr) repeat(3, minmax(0, 1fr)); gap: 12px; margin-bottom: 16px; }
.source-control { position: relative; display: flex; flex-direction: column; justify-content: space-between; align-items: flex-start; gap: 8px; min-width: 0; min-height: 70px; margin: 0; padding: 10px 12px; border: 1px solid var(--line); border-radius: 4px; color: var(--ink); font-size: 13px; line-height: 1.5; cursor: pointer; }
.source-control > span:first-child { color: var(--ink-2); }
.source-control select { width: 100%; min-width: 0; margin: 0; padding: 0; border: 0; background: var(--paper); color: var(--ink); font: inherit; font-size: 15px; cursor: pointer; }
.source-control input { position: absolute; width: 1px; height: 1px; margin: 0; opacity: 0; }
.source-toggle { position: relative; width: 32px; height: 18px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); }
.source-toggle::after { position: absolute; top: 3px; left: 3px; width: 10px; height: 10px; border-radius: 50%; background: var(--ink-2); content: ''; transition: transform 120ms; }
.source-control input:checked + .source-toggle { border-color: var(--blue); background: var(--blue); }
.source-control input:checked + .source-toggle::after { background: var(--on-accent); transform: translateX(14px); }
.source-control input:focus-visible + .source-toggle { outline: 2px solid var(--blue); outline-offset: 3px; }
.source-disabled { opacity: .55; cursor: default; }
.source-note, .copy-status { margin: 10px 0; color: var(--ink); font-size: 14px; line-height: 1.7; }
.source-code { min-width: 0; overflow: hidden; border: 1px solid var(--line); border-radius: 4px; background: var(--surface); }
.source-code-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 7px 10px 0 14px; color: var(--ink-2); font: 12px/1.5 ui-monospace, Consolas, monospace; }
.source-code-heading > span { min-width: 0; overflow-wrap: anywhere; }
.source-code-heading button { display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 3px; color: var(--ink); }
.source-code-heading button:hover { color: var(--blue); background: var(--blue-soft); }
.source-code pre { max-width: 100%; overflow: auto; margin: 0; padding: 6px 14px 16px; color: var(--ink); font: 13px/1.85 ui-monospace, 'Cascadia Code', Consolas, monospace; white-space: pre; }
@media (max-width: 1100px) { .source-controls { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 640px) {
  .source-controls { gap: 10px; }
  .source-control { min-height: 80px; padding: 10px; }
  .source-control select { min-height: 28px; font-size: 14px; }
  .source-code-heading button { width: 44px; height: 44px; }
  .source-code pre { font-size: 12px; }
}
@media (max-width: 360px) { .source-controls { grid-template-columns: minmax(0, 1fr); } }
@media (prefers-reduced-motion: reduce) { .source-toggle::after { transition: none; } }
</style>
