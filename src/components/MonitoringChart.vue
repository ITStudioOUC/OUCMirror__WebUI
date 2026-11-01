<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { MonitoringPanel } from '../monitoring'

const props = defineProps<{ panel: MonitoringPanel; timestamps: number[] }>()
const { t, d, n } = useI18n({ useScope: 'global' })
const plot = ref<HTMLElement>()
const width = ref(560)
const height = ref(216)
const hidden = ref<string[]>([])
const inspected = ref<number | null>(null)
const keyboardInspection = ref(false)
const left = 66
const right = 12
const top = 12
const bottom = 26
const plotWidth = computed(() => Math.max(1, width.value - left - right))
const plotHeight = computed(() => height.value - top - bottom)
const x = (index: number) => left + index / Math.max(1, props.timestamps.length - 1) * plotWidth.value
const y = (value: number) => top + (1 - value / props.panel.maximum) * plotHeight.value
function format(value: number) {
  const unit = props.panel.unit
  if (value > 0 && value < 1 && (unit === 'gigabits' || unit === 'gigabytes')) {
    return t(`monitor.units.${unit === 'gigabits' ? 'megabits' : 'megabytes'}`, { value: n(value * 1000, 'size') })
  }
  return t(`monitor.units.${unit}`, { value: n(value, 'size') })
}
const yTicks = computed(() => Array.from({ length: 5 }, (_, i) => props.panel.maximum * i / 4))
const timeTicks = computed(() => {
  const count = width.value < 420 ? 3 : 5
  return Array.from({ length: count }, (_, i) => Math.round((props.timestamps.length - 1) * i / (count - 1)))
})
const visible = computed(() => props.panel.series.filter(series => !hidden.value.includes(series.id)))
const paths = computed(() => {
  let baseline = props.timestamps.map(() => 0)
  return visible.value.map(series => {
    const base = [...baseline]
    const values = series.values.map((value, i) => value + (props.panel.stacked ? base[i]! : 0))
    if (props.panel.stacked) baseline = values
    const line = values.map((value, i) => `${i ? 'L' : 'M'}${x(i).toFixed(2)},${y(value).toFixed(2)}`).join(' ')
    const floor = base.map((value, i) => `L${x(i).toFixed(2)},${y(props.panel.stacked ? value : 0).toFixed(2)}`).reverse().join(' ')
    return { ...series, line, area: `${line} ${floor} Z`, plottedValues: values }
  })
})
const stats = computed(() => props.panel.series.map(series => ({
  ...series,
  mean: series.values.reduce((sum, value) => sum + value, 0) / Math.max(1, series.values.length),
  latest: series.values.at(-1) ?? 0,
  max: Math.max(0, ...series.values),
})))
const sampleIndex = computed(() => inspected.value === null ? null : Math.min(inspected.value, props.timestamps.length - 1))
const tooltipLeft = computed(() => `${Math.max(4, Math.min(x(sampleIndex.value ?? 0) + 12, width.value - 206))}px`)
function toggleSeries(id: string) {
  hidden.value = hidden.value.includes(id) ? hidden.value.filter(value => value !== id) : [...hidden.value, id]
}
function inspectPointer(event: PointerEvent) {
  keyboardInspection.value = false
  const bounds = (event.currentTarget as SVGElement).getBoundingClientRect()
  const position = Math.max(0, Math.min(1, (event.clientX - bounds.left - left) / plotWidth.value))
  inspected.value = Math.round(position * (props.timestamps.length - 1))
}
function leavePointer(event: PointerEvent) {
  if (event.pointerType !== 'touch') inspected.value = null
}
function onKeydown(event: KeyboardEvent) {
  const current = sampleIndex.value ?? props.timestamps.length - 1
  const last = props.timestamps.length - 1
  if (event.key === 'ArrowLeft') inspected.value = Math.max(0, current - 1)
  else if (event.key === 'ArrowRight') inspected.value = Math.min(last, current + 1)
  else if (event.key === 'Home') inspected.value = 0
  else if (event.key === 'End') inspected.value = last
  else if (event.key === 'Escape') inspected.value = null
  else return
  keyboardInspection.value = true
  event.preventDefault()
}
let observer: ResizeObserver | undefined
onMounted(() => {
  observer = new ResizeObserver(([entry]) => {
    if (!entry) return
    width.value = entry.contentRect.width
    height.value = entry.contentRect.height
  })
  if (plot.value) observer.observe(plot.value)
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <article class="monitor-chart" :data-panel="panel.id">
    <h3>{{ t(panel.titleKey) }}</h3>
    <div ref="plot" class="monitor-plot">
      <svg :viewBox="`0 0 ${width} ${height}`" role="img" :aria-label="t('monitor.chartHelp', { title: t(panel.titleKey) })" tabindex="0" @pointermove="inspectPointer" @pointerdown="inspectPointer" @pointerleave="leavePointer" @pointercancel="inspected = null" @blur="inspected = null" @keydown="onKeydown">
        <g v-for="tick in yTicks" :key="tick" class="chart-grid">
          <line :x1="left" :x2="width - right" :y1="y(tick)" :y2="y(tick)" />
          <text :x="left - 8" :y="y(tick) + 3" text-anchor="end">{{ format(tick) }}</text>
        </g>
        <g v-for="(tick, i) in timeTicks" :key="tick" class="chart-grid">
          <line :x1="x(tick)" :x2="x(tick)" :y1="top" :y2="height - bottom" />
          <text :x="x(tick)" :y="height - 7" :text-anchor="i === 0 ? 'start' : i === timeTicks.length - 1 ? 'end' : 'middle'">{{ d(timestamps[tick]!, 'monitorTick') }}</text>
        </g>
        <g v-for="series in paths" :key="series.id" :data-series="series.id">
          <path :d="series.area" :fill="series.color" :fill-opacity="panel.stacked ? .18 : .08" />
          <path :d="series.line" fill="none" :stroke="series.color" stroke-width="1.4" stroke-linejoin="round" />
        </g>
        <g v-if="sampleIndex !== null" class="chart-cursor">
          <line :x1="x(sampleIndex)" :x2="x(sampleIndex)" :y1="top" :y2="height - bottom" />
          <circle v-for="series in paths" :key="series.id" :cx="x(sampleIndex)" :cy="y(series.plottedValues[sampleIndex]!)" r="3" :fill="series.color" />
        </g>
      </svg>
      <div v-if="sampleIndex !== null" class="chart-tooltip" :style="{ left: tooltipLeft }" aria-hidden="true">
        <time :datetime="new Date(timestamps[sampleIndex]!).toISOString()">{{ d(timestamps[sampleIndex]!, 'monitorSample') }}</time>
        <div v-for="series in visible" :key="series.id"><span><i :style="{ background: series.color }" />{{ t(series.labelKey) }}</span><strong>{{ format(series.values[sampleIndex]!) }}</strong></div>
      </div>
      <p class="sr-only" aria-live="polite"><template v-if="keyboardInspection && sampleIndex !== null">{{ d(timestamps[sampleIndex]!, 'monitorSample') }}<span v-for="series in visible" :key="series.id"> {{ t(series.labelKey) }} {{ format(series.values[sampleIndex]!) }};</span></template></p>
    </div>
    <table class="chart-legend">
      <thead><tr><th scope="col">{{ t('monitor.legend.name') }}</th><th scope="col" class="chart-mean">{{ t('monitor.legend.mean') }}</th><th scope="col">{{ t('monitor.legend.latest') }}</th><th scope="col">{{ t('monitor.legend.max') }}</th></tr></thead>
      <tbody><tr v-for="series in stats" :key="series.id" :class="{ 'series-hidden': hidden.includes(series.id) }">
        <th scope="row"><button type="button" :aria-pressed="!hidden.includes(series.id)" :aria-label="t('monitor.toggleSeries', { name: t(series.labelKey) })" @click="toggleSeries(series.id)"><i :style="{ background: series.color }" />{{ t(series.labelKey) }}</button></th>
        <td class="chart-mean">{{ format(series.mean) }}</td><td>{{ format(series.latest) }}</td><td>{{ format(series.max) }}</td>
      </tr></tbody>
    </table>
  </article>
</template>

<style scoped>
.monitor-chart { --chart-muted: color-mix(in srgb, var(--ink) 38%, var(--panel)); min-width: 0; padding: 16px 14px 10px; border: 1px solid var(--line); border-radius: 6px; background: var(--panel); }
h3 { margin: 0 0 10px; font-size: 14px; font-weight: 600; line-height: 1.5; }
.monitor-plot { position: relative; height: 216px; }
svg { display: block; width: 100%; height: 100%; overflow: visible; touch-action: pan-y; }
.chart-grid line { stroke: var(--line); stroke-opacity: .45; stroke-width: .5; }
.chart-grid text { fill: color-mix(in srgb, var(--ink) 75%, var(--panel)); font-size: 10px; font-variant-numeric: tabular-nums; }
.chart-cursor line { stroke: var(--ink); stroke-opacity: .6; stroke-width: 1; stroke-dasharray: 3 3; }
.chart-cursor circle { stroke: var(--panel); stroke-width: 1.5; }
.chart-tooltip { position: absolute; z-index: 2; top: 8px; width: 198px; max-width: calc(100% - 8px); padding: 10px 12px; border: 1px solid var(--line); border-radius: 5px; background: var(--panel); color: var(--ink); box-shadow: var(--shadow); font-size: 11px; pointer-events: none; }
.chart-tooltip time { display: block; margin-bottom: 7px; font-variant-numeric: tabular-nums; }
.chart-tooltip div, .chart-tooltip span { display: flex; align-items: center; }
.chart-tooltip div { justify-content: space-between; gap: 8px; line-height: 1.9; }
.chart-tooltip span { gap: 6px; }
.chart-tooltip strong { font-weight: 500; font-variant-numeric: tabular-nums; }
i { display: inline-block; flex: 0 0 auto; width: 10px; height: 3px; }
.chart-legend { width: 100%; margin-top: 8px; border-collapse: collapse; font-size: 11px; font-variant-numeric: tabular-nums; }
.chart-legend th, .chart-legend td { padding: 4px 0 4px 6px; border-top: 1px solid var(--line-soft); text-align: right; white-space: nowrap; font-weight: 400; }
.chart-legend thead th { color: color-mix(in srgb, var(--ink) 70%, var(--panel)); font-size: 10px; }
.chart-legend th:first-child { padding-left: 0; text-align: left; }
.chart-legend button { display: inline-flex; align-items: center; gap: 7px; min-height: 24px; padding: 0; text-align: left; font-size: inherit; }
.chart-legend button:hover { color: var(--blue); }
.series-hidden { opacity: .45; }
@media (max-width: 700px) {
  .monitor-chart { padding: 13px 10px 8px; }
  .monitor-plot { height: 200px; }
  .chart-legend button { min-height: 36px; }
}
@media (max-width: 480px) { .chart-mean { display: none; } }
</style>
