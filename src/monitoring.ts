export type MonitoringRange = 1 | 6 | 24
export type MetricUnit = 'gigabits' | 'gigabytes' | 'operations' | 'percent' | 'memory'
export type MetricValue = number | null
export interface MetricSeries { id: string; labelKey: string; color: string; values: MetricValue[]; stats?: { mean: number; latest: MetricValue; max: number } }
export interface MonitoringPanel {
  id: string
  titleKey: string
  unit: MetricUnit
  maximum: number
  stacked: boolean
  series: MetricSeries[]
  available?: boolean
}
export interface MonitoringSnapshot { timestamps: number[]; panels: MonitoringPanel[] }
export interface LiveMonitoringSnapshot extends MonitoringSnapshot { disks: { name: string; used: number; total: number }[]; updatedAt: number; stale: boolean }

export function metricStatistics(values: MetricValue[]) {
  const measured = values.filter((value): value is number => value !== null && Number.isFinite(value))
  const latest = values.at(-1)
  return {
    mean: measured.reduce((sum, value) => sum + value, 0) / Math.max(1, measured.length),
    latest: latest != null && Number.isFinite(latest) ? latest : null,
    max: Math.max(0, ...measured),
  }
}

// Missing measurements break both lines and fills; they are never plotted as zero.
export function metricPaths(series: MetricSeries[], count: number, stacked: boolean, coordinate: (index: number, value: number) => string) {
  let baseline: MetricValue[] = Array.from({ length: count }, () => 0)
  return series.map(item => {
    const base = baseline
    const values = Array.from({ length: count }, (_, index) => {
      const value = item.values[index]
      if (value == null || !Number.isFinite(value) || (stacked && base[index] == null)) return null
      return value + (stacked ? base[index]! : 0)
    })
    if (stacked) baseline = values
    const segments: number[][] = []
    let segment: number[] = []
    for (let index = 0; index <= count; index++) {
      if (index < count && values[index] != null) segment.push(index)
      else if (segment.length) { segments.push(segment); segment = [] }
    }
    const lineFor = (indices: number[]) => indices.map((index, position) => `${position ? 'L' : 'M'}${coordinate(index, values[index]!)}`).join(' ')
    return {
      ...item,
      line: segments.map(lineFor).join(' '),
      area: segments.map(indices => `${lineFor(indices)} ${[...indices].reverse().map(index => `L${coordinate(index, stacked ? base[index]! : 0)}`).join(' ')} Z`).join(' '),
      plottedValues: values,
    }
  })
}

export const demoDisks = [
  { name: '/srv/mirrors', used: 43.2, total: 64 },
  { name: '/srv/archive', used: 12.8, total: 32 },
]

// Deterministic, continuous signals keep range changes and refreshes consistent.
// These are UI fixtures, not measurements from the mirror server.
function noise(seconds: number, seed: number, period = 90) {
  const position = seconds / period
  const bucket = Math.floor(position)
  const hash = (value: number) => {
    const result = Math.sin(value * 127.1 + seed * 311.7) * 43758.5453
    return result - Math.floor(result)
  }
  const fraction = position - bucket
  const eased = fraction * fraction * (3 - 2 * fraction)
  return hash(bucket) * (1 - eased) + hash(bucket + 1) * eased
}

export function createMonitoringSnapshot(end: number, hours: MonitoringRange): MonitoringSnapshot {
  const timestamps = Array.from({ length: 145 }, (_, i) => end - hours * 3_600_000 * (1 - i / 144))
  const samples = timestamps.map(timestamp => {
    const seconds = timestamp / 1000
    const hour = (seconds / 3600 + 8) % 24
    const activity = .55 + .35 * Math.cos((hour - 14) * Math.PI / 12)
    const n = (seed: number, period?: number) => noise(seconds, seed, period)
    const user = 10 + activity * 25 + n(9) * 6
    const system = 3 + activity * 9 + n(10) * 3
    const wait = .4 + Math.pow(n(11), 5) * 6
    const used = 74 + activity * 7 + n(12, 600) * 2
    const cached = 22 + n(13, 600) * 3
    const buffers = 2 + n(14, 600)
    return {
      https4: 1.4 + activity * 4 + n(2) * 1.1,
      rsync4: .2 + activity * .6 + Math.pow(n(3), 3) * .9,
      https6: .4 + activity * 1.4 + n(5) * .4,
      rsync6: .1 + activity * .3 + Math.pow(n(6), 3) * .4,
      readOps: 8 + activity * 34 + n(7) * 10,
      writeOps: .5 + n(8) * 1.5 + Math.pow(n(15, 45), 12) * 15,
      readBytes: .4 + activity * 1.5 + n(16) * .45,
      writeBytes: .012 + n(17) * .025 + Math.pow(n(18, 45), 12) * .18,
      user, system, wait, idle: 100 - user - system - wait,
      used, cached, buffers, free: 128 - used - cached - buffers,
    }
  })
  type SampleKey = keyof typeof samples[number]
  const series = (id: SampleKey, label: string, color: string): MetricSeries => ({
    id, labelKey: `monitor.series.${label}`, color: `var(--${color})`, values: samples.map(sample => sample[id]),
  })
  const panel = (id: string, unit: MetricUnit, maximum: number, stacked: boolean, series: MetricSeries[]): MonitoringPanel => ({
    id, titleKey: `monitor.charts.${id}`, unit, maximum, stacked, series,
  })
  return { timestamps, panels: [
    panel('ipv4', 'gigabits', 10, true, [series('https4', 'https', 'blue'), series('rsync4', 'rsync', 'violet')]),
    panel('ipv6', 'gigabits', 4, true, [series('https6', 'https', 'blue'), series('rsync6', 'rsync', 'violet')]),
    panel('iops', 'operations', 60, false, [series('readOps', 'read', 'green'), series('writeOps', 'write', 'amber')]),
    panel('throughput', 'gigabytes', 3, false, [series('readBytes', 'read', 'blue'), series('writeBytes', 'write', 'violet')]),
    panel('cpu', 'percent', 100, true, [series('user', 'user', 'blue'), series('system', 'system', 'green'), series('wait', 'wait', 'amber'), series('idle', 'idle', 'chart-muted')]),
    panel('memory', 'memory', 128, true, [series('used', 'used', 'blue'), series('cached', 'cached', 'green'), series('buffers', 'buffers', 'amber'), series('free', 'free', 'chart-muted')]),
  ] }
}
