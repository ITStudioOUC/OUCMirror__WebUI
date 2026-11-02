package main

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

type metricsHistory struct {
	UpdatedAt   int64                `json:"updatedAt"`
	MemoryTotal float64              `json:"memoryTotal"`
	Disks       []map[string]any     `json:"disks"`
	Samples     []map[string]float64 `json:"samples"`
}
type metricStats struct {
	Mean   float64  `json:"mean"`
	Latest *float64 `json:"latest"`
	Max    float64  `json:"max"`
}
type metricSeries struct {
	ID       string      `json:"id"`
	LabelKey string      `json:"labelKey"`
	Color    string      `json:"color"`
	Values   []*float64  `json:"values"`
	Stats    metricStats `json:"stats"`
}
type metricPanel struct {
	ID        string         `json:"id"`
	TitleKey  string         `json:"titleKey"`
	Unit      string         `json:"unit"`
	Maximum   float64        `json:"maximum"`
	Stacked   bool           `json:"stacked"`
	Available bool           `json:"available"`
	Series    []metricSeries `json:"series"`
}

func niceMaximum(value float64) float64 {
	if value <= 0 {
		return 1
	}
	power := math.Pow(10, math.Floor(math.Log10(value)))
	for _, step := range []float64{1, 2, 5, 10} {
		if step*power >= value*1.05 {
			return step * power
		}
	}
	return 20 * power
}
func (a *App) monitoring(w http.ResponseWriter, r *http.Request) {
	hours, err := strconv.Atoi(r.URL.Query().Get("hours"))
	if err != nil || (hours != 1 && hours != 6 && hours != 24) {
		fail(w, 400, "INVALID_RANGE")
		return
	}
	var history metricsHistory
	if a.cfg.Metrics == "" {
		fail(w, 503, "METRICS_UNAVAILABLE")
		return
	}
	raw, err := os.ReadFile(a.cfg.Metrics)
	if err != nil || len(raw) > 32<<20 {
		fail(w, 503, "METRICS_UNAVAILABLE")
		return
	}
	if err = json.Unmarshal(raw, &history); err != nil {
		fail(w, 503, "METRICS_UNAVAILABLE")
		return
	}
	samples := []map[string]float64{}
	cutoff := float64(time.Now().Add(-time.Duration(hours) * time.Hour).UnixMilli())
	for _, sample := range history.Samples {
		if sample["time"] >= cutoff {
			samples = append(samples, sample)
		}
	}
	indices := []int{}
	if len(samples) <= 241 {
		for i := range samples {
			indices = append(indices, i)
		}
	} else {
		for i := 0; i < 241; i++ {
			indices = append(indices, int(math.Round(float64(i)*float64(len(samples)-1)/240)))
		}
	}
	timestamps := []int64{}
	for _, index := range indices {
		timestamps = append(timestamps, int64(samples[index]["time"]))
	}
	makeSeries := func(id, label, color string) metricSeries {
		series := metricSeries{ID: id, LabelKey: "monitor.series." + label, Color: "var(--" + color + ")", Values: []*float64{}}
		count := 0
		for _, sample := range samples {
			if value, ok := sample[id]; ok {
				series.Stats.Mean += value
				series.Stats.Max = math.Max(value, series.Stats.Max)
				count++
			}
		}
		if count > 0 {
			series.Stats.Mean /= float64(count)
		}
		if len(samples) > 0 {
			if latest, ok := samples[len(samples)-1][id]; ok {
				series.Stats.Latest = &latest
			}
		}
		previousIndex := -1
		for _, index := range indices {
			var point *float64
			complete := true
			// Preserve gaps even when downsampling skips the original missing point.
			for i := previousIndex + 1; i <= index; i++ {
				if _, ok := samples[i][id]; !ok {
					complete = false
					break
				}
			}
			if complete {
				value := samples[index][id]
				point = &value
			}
			series.Values = append(series.Values, point)
			previousIndex = index
		}
		return series
	}
	panel := func(id, unit string, maximum float64, stacked bool, series ...metricSeries) metricPanel {
		completePoints := 0
		peak := float64(0)
		for _, sample := range samples {
			sum := float64(0)
			complete := true
			for _, s := range series {
				value, ok := sample[s.ID]
				if !ok {
					complete = false
				}
				if stacked {
					sum += value
				} else {
					sum = math.Max(sum, value)
				}
			}
			peak = math.Max(peak, sum)
			if complete {
				completePoints++
			}
		}
		if maximum == 0 {
			maximum = niceMaximum(peak)
		}
		if maximum <= 0 {
			maximum = 1
		}
		return metricPanel{id, "monitor.charts." + id, unit, maximum, stacked, completePoints > 1, series}
	}
	panels := []metricPanel{
		panel("ipv4", "gigabits", 0, false, makeSeries("http4", "http", "blue"), makeSeries("rsync4", "rsync", "violet")),
		panel("ipv6", "gigabits", 0, false, makeSeries("http6", "http", "blue"), makeSeries("rsync6", "rsync", "violet")),
		panel("iops", "operations", 0, false, makeSeries("readOps", "read", "green"), makeSeries("writeOps", "write", "amber")),
		panel("throughput", "gigabytes", 0, false, makeSeries("readBytes", "read", "blue"), makeSeries("writeBytes", "write", "violet")),
		panel("cpu", "percent", 100, true, makeSeries("user", "user", "blue"), makeSeries("system", "system", "green"), makeSeries("wait", "wait", "amber"), makeSeries("idle", "idle", "chart-muted")),
		panel("memory", "memory", history.MemoryTotal, true, makeSeries("used", "used", "blue"), makeSeries("cached", "cached", "green"), makeSeries("buffers", "buffers", "amber"), makeSeries("free", "free", "chart-muted")),
	}
	respond(w, 200, map[string]any{"timestamps": timestamps, "panels": panels, "disks": history.Disks, "updatedAt": history.UpdatedAt, "stale": time.Now().UnixMilli()-history.UpdatedAt > 90000})
}
