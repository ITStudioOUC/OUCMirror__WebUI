<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Pause, Play, Search, X } from 'lucide-vue-next'
import type { Mirror } from '../data'
import { type MonitoringRange, type LiveMonitoringSnapshot } from '../monitoring'
import { api } from '../api'
import { useMirrorFormats } from '../useMirrorFormats'
import MonitoringChart from './MonitoringChart.vue'
import RollingText from './RollingText.vue'

const props = defineProps<{ mirrors: Mirror[]; now: number; cycle: number; showNextSync: boolean }>()
const emit = defineEmits<{ mirror: [mirror: Mirror] }>()
const { t, n } = useI18n({ useScope: 'global' })
const { mirrorSize, relativeTime, fullTime } = useMirrorFormats(toRef(props, 'now'))
const range = ref<MonitoringRange>(24)
const ranges: { hours: MonitoringRange; key: string }[] = [
  { hours: 1, key: 'one' }, { hours: 6, key: 'six' }, { hours: 24, key: 'day' },
]
const refreshing = ref(true)
const snapshot = ref<LiveMonitoringSnapshot>({ timestamps: [], panels: [], disks: [], updatedAt: 0, stale: false })
const snapshotAt = computed(() => snapshot.value.updatedAt)
const loading = ref(true)
const failed = ref(false)
interface SpeedSnapshot { updatedAt: number; interval: number; stale: boolean; available: boolean; jobs: Record<string, { bytesPerSecond: number; connections: number }> }
const speeds = ref<SpeedSnapshot | null>(null)
const speedLoading = ref(true)
let speedController: AbortController | undefined
async function refreshSpeeds() {
  speedController?.abort(); speedController = new AbortController()
  try { speeds.value = await api<SpeedSnapshot>('/job-speeds', { signal: speedController.signal }) }
  catch (error) { if ((error as Error).name !== 'AbortError') speeds.value = null }
  finally { speedLoading.value = false }
}
function pullSpeed(mirror: Mirror) {
  if (mirror.status !== 'syncing') return t('common.emptyValue')
  if (speedLoading.value || (speeds.value && !speeds.value.available && !speeds.value.stale)) return t('monitor.speedCollecting')
  const sample = speeds.value?.jobs[mirror.name]
  if (!sample || speeds.value?.stale || props.now - speeds.value!.updatedAt > 15000) return t('common.emptyValue')
  const bytes = sample.bytesPerSecond
  const unit = bytes >= 1e9 ? 'gigabytes' : bytes >= 1e6 ? 'megabytes' : bytes >= 1e3 ? 'kilobytes' : 'bytes'
  const scale = { gigabytes: 1e9, megabytes: 1e6, kilobytes: 1e3, bytes: 1 }[unit]
  return t(`monitor.speedUnits.${unit}`, { value: n(bytes / scale, 'size') })
}
let controller: AbortController | undefined
let sequence = 0
async function refresh() {
  const request = ++sequence
  controller?.abort(); controller = new AbortController()
  try {
    const data = await api<LiveMonitoringSnapshot>(`/monitoring?hours=${range.value}`, { signal: controller.signal })
    if (request === sequence) { snapshot.value = data; failed.value = false }
  } catch (error) { if ((error as Error).name !== 'AbortError' && request === sequence) failed.value = true }
  finally { if (request === sequence) loading.value = false }
}
watch(range, () => { void refresh() })
const search = ref('')
const searchInput = ref<HTMLInputElement>()
const filteredMirrors = computed(() => {
  const query = search.value.trim().toLowerCase()
  return props.mirrors.filter(mirror => `${mirror.name} ${mirror.upstream}`.toLowerCase().includes(query))
})
const capacity = (value: number) => t(value < 1 ? 'monitor.capacityGB' : 'monitor.units.capacity', { value: n(value < 1 ? value * 1000 : value, 'size') })
function syncInfo(mirror: Mirror, next: boolean) {
  if (next && mirror.nextSync !== null) return { key: 'next', timestamp: mirror.nextSync, next: true }
  if (mirror.status === 'syncing' && mirror.syncStarted !== null) return { key: 'started', timestamp: mirror.syncStarted, next: false }
  if (mirror.status === 'failed' && mirror.lastAttempt !== null) return { key: 'attempt', timestamp: mirror.lastAttempt, next: false }
  return { key: 'success', timestamp: mirror.lastSuccess, next: false }
}
function syncLabel(mirror: Mirror, next: boolean) { return t(`monitor.syncTime.${syncInfo(mirror, next).key}`) }
function syncTime(mirror: Mirror, next: boolean) {
  const info = syncInfo(mirror, next)
  return relativeTime(info.timestamp, info.next)
}
function focusSearch() { searchInput.value?.focus() }
defineExpose({ focusSearch })
let timer: ReturnType<typeof setInterval> | undefined
let speedTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  void refresh()
  void refreshSpeeds()
  speedTimer = setInterval(() => { if (refreshing.value && !document.hidden) void refreshSpeeds() }, 2000)
  timer = setInterval(() => {
    if (refreshing.value && !document.hidden) void refresh()
  }, 10000)
})
onBeforeUnmount(() => { sequence++; controller?.abort(); speedController?.abort(); clearInterval(timer); clearInterval(speedTimer) })
</script>

<template>
  <section id="site-status" class="status-page" tabindex="-1" :data-range="range" :data-updated="snapshotAt" :data-refreshing="refreshing">
    <div class="section-heading monitor-heading">
      <div class="monitor-title"><h1>{{ t('monitor.title') }}</h1><span class="monitor-demo">{{ t(failed || snapshot.stale ? 'monitor.stale' : 'monitor.live') }}</span></div>
      <div class="monitor-controls">
        <div class="monitor-ranges" role="group" :aria-label="t('monitor.range')"><button v-for="choice in ranges" :key="choice.hours" type="button" :aria-pressed="range === choice.hours" @click="range = choice.hours">{{ t(`monitor.ranges.${choice.key}`) }}</button></div>
        <button type="button" class="monitor-refresh" @click="refreshing = !refreshing; if (refreshing) refresh()"><Pause v-if="refreshing" :size="14" aria-hidden="true" /><Play v-else :size="14" aria-hidden="true" />{{ t(refreshing ? 'monitor.pauseRefresh' : 'monitor.resumeRefresh') }}</button>
      </div>
    </div>
    <p v-if="loading || failed" role="status">{{ t(loading ? 'monitor.loading' : 'monitor.unavailable') }}</p>

    <section class="monitor-section" aria-labelledby="monitor-network">
      <div class="side-heading monitor-section-heading"><h2 id="monitor-network">{{ t('monitor.network') }}</h2></div>
      <div class="monitor-grid"><MonitoringChart v-for="panel in snapshot.panels.slice(0, 2)" :key="panel.id" :panel="panel" :timestamps="snapshot.timestamps" /></div>
    </section>

    <section class="monitor-section" aria-labelledby="monitor-disk">
      <div class="side-heading monitor-section-heading"><h2 id="monitor-disk">{{ t('monitor.disk') }}</h2></div>
      <div class="disk-capacities" role="group" :aria-label="t('monitor.capacity')">
        <div v-for="disk in snapshot.disks" :key="disk.name" class="disk-capacity">
          <div class="disk-caption"><span>{{ disk.name }}</span><span>{{ t('monitor.diskUsage', { used: capacity(disk.used), total: capacity(disk.total) }) }}</span></div>
          <div class="disk-track" role="meter" :aria-label="t('monitor.diskDescription', { name: disk.name, used: capacity(disk.used), total: capacity(disk.total) })" :aria-valuemin="0" :aria-valuemax="disk.total" :aria-valuenow="disk.used"><span :style="{ width: `${disk.used / disk.total * 100}%` }" /></div>
        </div>
      </div>
      <div class="monitor-grid"><MonitoringChart v-for="panel in snapshot.panels.slice(2, 4)" :key="panel.id" :panel="panel" :timestamps="snapshot.timestamps" /></div>
    </section>

    <section class="monitor-section" aria-labelledby="monitor-resources">
      <div class="side-heading monitor-section-heading"><h2 id="monitor-resources">{{ t('monitor.resources') }}</h2></div>
      <div class="monitor-grid"><MonitoringChart v-for="panel in snapshot.panels.slice(4)" :key="panel.id" :panel="panel" :timestamps="snapshot.timestamps" /></div>
    </section>

    <section class="monitor-section" aria-labelledby="monitor-sync">
      <div class="side-heading monitor-section-heading sync-heading">
        <h2 id="monitor-sync">{{ t('monitor.sync') }}</h2>
        <div class="search-box sync-search"><Search :size="17" aria-hidden="true" /><input ref="searchInput" v-model="search" type="search" :placeholder="t('monitor.search')" :aria-label="t('monitor.search')" autocomplete="off" /><button v-if="search" class="icon-button clear-search" :aria-label="t('mirrors.clear')" @click="search = ''; focusSearch()"><X :size="15" /></button></div>
      </div>
      <table class="sync-table" role="table" aria-labelledby="monitor-sync">
        <thead role="rowgroup"><tr role="row"><th v-for="column in ['name', 'update', 'upstream', 'status', 'size', 'speed']" :key="column" role="columnheader" scope="col" :data-column="column" :title="column === 'speed' ? t('monitor.speedDescription') : undefined">{{ t(`monitor.columns.${column}`) }}</th></tr></thead>
        <tbody role="rowgroup"><tr v-for="mirror in filteredMirrors" :key="mirror.name" role="row" :data-status="mirror.status">
          <td role="cell" data-column="name"><button class="sync-name" :aria-label="t('mirrors.browse', { name: mirror.name })" @click="emit('mirror', mirror)">{{ mirror.name }}</button></td>
          <td role="cell" data-column="update" :title="fullTime(syncInfo(mirror, showNextSync).timestamp)"><div class="sync-update"><RollingText :value="syncLabel(mirror, showNextSync)" :cycle="cycle" :reserve="[syncLabel(mirror, false), syncLabel(mirror, true)]" :shuffle="false" /><RollingText :value="syncTime(mirror, showNextSync)" :cycle="cycle" :reserve="[syncTime(mirror, false), syncTime(mirror, true)]" :shuffle="false" /></div></td>
          <td role="cell" data-column="upstream"><span class="sync-mobile-label" aria-hidden="true">{{ t('monitor.columns.upstream') }}</span><a :href="mirror.upstream" target="_blank" rel="noopener noreferrer">{{ mirror.upstream }}</a></td>
          <td role="cell" data-column="status">{{ t(`mirrors.status.${mirror.status}`) }}</td>
          <td role="cell" data-column="size"><span class="sync-mobile-label" aria-hidden="true">{{ t('monitor.columns.size') }}</span><span>{{ mirrorSize(mirror) }}</span></td>
          <td role="cell" data-column="speed" :title="t('monitor.speedDescription')"><span class="sync-mobile-label" aria-hidden="true">{{ t('monitor.columns.speed') }}</span><span>{{ pullSpeed(mirror) }}</span></td>
        </tr></tbody>
      </table>
      <div v-if="!filteredMirrors.length" class="sync-empty"><p>{{ t('mirrors.empty') }}</p><button class="text-button" @click="search = ''; focusSearch()">{{ t('mirrors.reset') }}</button></div>
    </section>
  </section>
</template>

<style scoped>
.monitor-heading { align-items: center; flex-wrap: wrap; gap: 18px 24px; margin-bottom: 30px; }
.monitor-title { display: flex; align-items: baseline; flex-wrap: wrap; gap: 12px; }
.monitor-demo { color: var(--ink-2); font-size: 12px; white-space: nowrap; }
.monitor-controls { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.monitor-ranges { display: inline-flex; gap: 2px; padding: 3px; border: 1px solid var(--line); border-radius: 6px; }
.monitor-ranges button { min-height: 30px; padding: 5px 10px; border-radius: 3px; color: var(--ink); font-size: 12px; white-space: nowrap; }
.monitor-ranges button:hover, .monitor-ranges button[aria-pressed='true'] { background: var(--blue-soft); color: var(--blue); }
.monitor-ranges button[aria-pressed='true'] { font-weight: 600; }
.monitor-refresh { display: inline-flex; align-items: center; gap: 6px; min-height: 38px; padding: 7px 10px; border: 1px solid var(--line); border-radius: 6px; color: var(--ink); font-size: 12px; }
.monitor-refresh:hover { color: var(--blue); background: var(--blue-soft); }
.monitor-section + .monitor-section { margin-top: 32px; }
.monitor-section-heading { gap: 16px; margin-bottom: 16px; }
.monitor-section-heading h2 { font-size: 26px; line-height: 1.3; letter-spacing: -.02em; }
.monitor-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 24px; }
.disk-capacities { display: grid; gap: 16px; margin-bottom: 24px; }
.disk-caption { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 6px 16px; margin-bottom: 7px; font-size: 13px; font-variant-numeric: tabular-nums; }
.disk-caption > span:first-child { font-family: ui-monospace, Consolas, monospace; }
.disk-track { height: 12px; overflow: hidden; background: var(--line); border-radius: 3px; }
.disk-track span { display: block; height: 100%; background: var(--blue); }
.sync-search { width: 290px; max-width: 100%; height: 38px; }
.sync-search input { font-size: 12px; }
.sync-table { width: 100%; border-collapse: collapse; table-layout: fixed; font-size: 13px; }
.sync-table th { padding: 11px 10px; border-bottom: 2px solid var(--line); font-size: 14px; font-weight: 700; text-align: left; }
.sync-table th[data-column='name'] { width: 14%; }
.sync-table th[data-column='update'] { width: 26%; }
.sync-table th[data-column='upstream'] { width: 29%; }
.sync-table th[data-column='status'] { width: 10%; }
.sync-table th[data-column='size'] { width: 9%; }
.sync-table th[data-column='speed'] { width: 12%; }
.sync-table td { padding: 7px 10px; height: 46px; border-bottom: 1px solid var(--line); overflow-wrap: anywhere; }
.sync-table :is(th, td):first-child { padding-left: 0; }
.sync-table :is(th, td):last-child { padding-right: 0; }
.sync-table tr[data-status='syncing'] { background: var(--blue-soft); }
.sync-table tr[data-status='failed'] { background: var(--red-soft); }
.sync-name { padding: 0; min-height: 30px; color: var(--blue); font: 600 13px ui-monospace, Consolas, monospace; text-align: left; overflow-wrap: anywhere; }
.sync-name:hover, .sync-table a:hover { color: var(--blue); text-decoration: underline; text-underline-offset: 3px; }
.sync-update { display: grid; grid-template-columns: max-content minmax(0, 1fr); align-items: center; gap: 12px; font-size: 12px; font-variant-numeric: tabular-nums; }
.sync-table td[data-column='upstream'] { font-size: 12px; }
.sync-table td[data-column='size'] { font-variant-numeric: tabular-nums; white-space: nowrap; }
.sync-table td[data-column='speed'] { font-variant-numeric: tabular-nums; white-space: nowrap; font-size: 12px; }
.sync-mobile-label { display: none; }
.sync-empty { padding: 32px 0; text-align: center; color: var(--ink-2); font-size: 14px; }
@media (max-width: 980px) {
  .sync-table th[data-column='name'] { width: 16%; }
  .sync-table th[data-column='update'] { width: 22%; }
  .sync-table th[data-column='upstream'] { width: 27%; }
  .sync-table th[data-column='status'] { width: 12%; }
  .sync-table th[data-column='size'] { width: 10%; }
  .sync-table th[data-column='speed'] { width: 13%; }
  .sync-update { grid-template-columns: minmax(0, 1fr); gap: 2px; }
}
@media (max-width: 900px) { .monitor-grid { grid-template-columns: minmax(0, 1fr); gap: 18px; } }
@media (max-width: 700px) {
  .monitor-heading { margin-bottom: 24px; }
  .monitor-title { gap: 9px; }
  .monitor-controls { gap: 8px; }
  .monitor-ranges { gap: 0; padding: 0; }
  .monitor-ranges button { min-height: 44px; padding-inline: 9px; }
  .monitor-refresh { min-height: 44px; padding-inline: 8px; }
  .monitor-section + .monitor-section { margin-top: 28px; }
  .monitor-section-heading h2 { font-size: 22px; }
  .disk-caption { font-size: 12px; }
  .sync-heading { align-items: stretch; flex-direction: column; gap: 14px; }
  .sync-search { width: 100%; height: 44px; }
  .sync-search input { font-size: 16px; }
  .sync-table, .sync-table tbody { display: block; }
  .sync-table thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); }
  .sync-table tr { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 8px 12px; padding: 12px; border-bottom: 1px solid var(--line); }
  .sync-table td { min-width: 0; height: auto; padding: 0; border: 0; font-size: 13px; }
  .sync-table td[data-column='name'] { grid-column: 1; grid-row: 1; }
  .sync-table td[data-column='status'] { grid-column: 2; grid-row: 1; align-self: center; }
  .sync-table td[data-column='update'] { grid-column: 1 / -1; }
  .sync-table td[data-column='upstream'], .sync-table td[data-column='size'], .sync-table td[data-column='speed'] { display: grid; grid-column: 1 / -1; grid-template-columns: 64px minmax(0, 1fr); align-items: baseline; gap: 10px; font-size: 12px; }
  .sync-table td[data-column='upstream'] a { color: var(--blue); }
  .sync-name { min-height: 36px; font-size: 14px; }
  .sync-update { grid-template-columns: max-content minmax(0, 1fr); gap: 8px; }
  .sync-mobile-label { display: inline; color: var(--ink-2); }
}
</style>
