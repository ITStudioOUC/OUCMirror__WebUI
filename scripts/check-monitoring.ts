import assert from 'node:assert/strict'
import { createMonitoringSnapshot, demoDisks, type MonitoringRange } from '../src/monitoring.ts'

const end = Date.parse('2026-11-01T16:00:00+08:00')
for (const hours of [1, 6, 24] as MonitoringRange[]) {
  const snapshot = createMonitoringSnapshot(end, hours)
  assert.deepEqual(snapshot, createMonitoringSnapshot(end, hours), 'The same window must reproduce the same data')
  assert.equal(snapshot.timestamps.length, 145)
  assert.equal(snapshot.timestamps[0], end - hours * 3_600_000)
  assert.equal(snapshot.timestamps.at(-1), end)
  assert.equal(snapshot.panels.length, 6)
  for (const panel of snapshot.panels) {
    for (const series of panel.series) {
      assert.equal(series.values.length, snapshot.timestamps.length)
      assert.ok(series.values.every(value => Number.isFinite(value) && value >= 0 && value <= panel.maximum))
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
console.log('Monitoring check passed: deterministic windows, timestamps, valid samples, stack limits, CPU/memory totals and disk capacities.')
