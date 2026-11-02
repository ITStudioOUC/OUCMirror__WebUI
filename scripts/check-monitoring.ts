import assert from 'node:assert/strict'
import { createMonitoringSnapshot, demoDisks, metricPaths, metricStatistics, type MetricSeries, type MonitoringRange } from '../src/monitoring.ts'

const end = Date.parse('2026-11-01T16:00:00+08:00')
for (const hours of [1, 6, 24] as MonitoringRange[]) {
  const snapshot = createMonitoringSnapshot(end, hours)
  assert.deepEqual(snapshot, createMonitoringSnapshot(end, hours), 'The same window must reproduce the same data')
  assert.equal(snapshot.timestamps.length, 145)
  assert.equal(snapshot.timestamps[0], end - hours * 3_600_000)
  assert.equal(snapshot.timestamps.at(-1), end)
  assert.equal(snapshot.panels.length, 6)
  assert.equal(snapshot.panels[0]!.series[0]!.id, 'http4')
  assert.equal(snapshot.panels[1]!.series[0]!.id, 'http6')
  assert.equal(snapshot.panels[0]!.series[0]!.labelKey, 'monitor.series.http')
  for (const panel of snapshot.panels) {
    for (const series of panel.series) {
      assert.equal(series.values.length, snapshot.timestamps.length)
      assert.ok(series.values.every(value => value !== null && Number.isFinite(value) && value >= 0 && value <= panel.maximum))
    }
    for (let index = 0; index < snapshot.timestamps.length; index++) {
      if (index) assert.ok(snapshot.timestamps[index]! > snapshot.timestamps[index - 1]!)
      const sum = panel.series.reduce((total, series) => total + series.values[index]!, 0)
      if (panel.stacked) assert.ok(sum <= panel.maximum + 1e-8, `${panel.id} exceeds the axis limit`)
      if (panel.id === 'cpu' || panel.id === 'memory') assert.ok(Math.abs(sum - panel.maximum) < 1e-8, `${panel.id} must preserve its total`)
    }
  }
}
for (const disk of demoDisks) assert.ok(disk.used >= 0 && disk.used <= disk.total)
const series = (id: string, values: MetricSeries['values']): MetricSeries => ({ id, values, labelKey: id, color: 'blue' })
const lower = series('lower', [0, 2, null, 3, 4])
const upper = series('upper', [1, 1, 1, 1, 1])
const coordinate = (index: number, value: number) => `${index},${value}`
const paths = metricPaths([lower, upper], 5, true, coordinate)
assert.equal(paths[0]!.line, 'M0,0 L1,2 M3,3 L4,4', 'Missing data must break a line, preserving measured zero')
assert.equal(paths[0]!.area, 'M0,0 L1,2 L1,0 L0,0 Z M3,3 L4,4 L4,0 L3,0 Z', 'Fills must not bridge gaps')
assert.deepEqual(paths[1]!.plottedValues, [1, 3, null, 4, 5], 'Unknown stacked baselines must remain gaps')
assert.deepEqual(metricPaths([upper], 5, true, coordinate)[0]!.plottedValues, upper.values, 'Hiding a lower series must rebuild the stack')
assert.deepEqual(metricStatistics([null, 0, 4, null]), { mean: 2, latest: null, max: 4 }, 'Missing samples must not become zero or stale latest values')
assert.deepEqual(metricStatistics([null, null]), { mean: 0, latest: null, max: 0 })
for (const family of [4, 6]) {
  const panel = createMonitoringSnapshot(end, 24).panels[family === 4 ? 0 : 1]!
  panel.series = [series(`http${family}`, [null, null, null, .1, .2]), series(`rsync${family}`, [.2, .3, .4, .5, .6])]
  assert.equal(panel.stacked, false, 'Network protocols must use independent rates')
  const networkPaths = metricPaths(panel.series, 5, panel.stacked, coordinate)
  assert.deepEqual(networkPaths[0]!.plottedValues, [null, null, null, .1, .2], 'HTTP/S coordinates must match its rate, not the combined traffic')
  assert.deepEqual(networkPaths[1]!.plottedValues, [.2, .3, .4, .5, .6], 'New HTTP/S collection must not hide existing rsync history')
  assert.equal(networkPaths[1]!.line, 'M0,0.2 L1,0.3 L2,0.4 L3,0.5 L4,0.6')
  const httpOnly = metricPaths([panel.series[0]!], 5, panel.stacked, coordinate)
  assert.deepEqual(httpOnly[0]!.plottedValues, networkPaths[0]!.plottedValues, 'Hiding rsync must not move the HTTP/S curve')
}
console.log('Monitoring check passed: deterministic windows, statistics, missing samples, line/fill gaps, stack limits and disk capacities.')
