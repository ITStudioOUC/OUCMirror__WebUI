package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMonitoringRecoversAfterMissingSamples(t *testing.T) {
	c := fixture(t)
	path := filepath.Join(t.TempDir(), "history.json")
	c.app.cfg.Metrics = path
	now := time.Now()
	history := metricsHistory{UpdatedAt: now.UnixMilli(), MemoryTotal: 32}
	for i := 0; i < 500; i++ {
		sample := map[string]float64{"time": float64(now.Add(time.Duration(i-500) * time.Second).UnixMilli())}
		for _, key := range []string{"http4", "rsync4", "readOps", "writeOps", "readBytes", "writeBytes"} {
			sample[key] = 4
		}
		history.Samples = append(history.Samples, sample)
	}
	// A collector startup or counter reset must not disable an entire history window.
	for _, key := range []string{"http4", "rsync4", "readOps", "writeOps", "readBytes", "writeBytes"} {
		delete(history.Samples[0], key)
		delete(history.Samples[1], key) // Also exercise a gap skipped by downsampling.
		delete(history.Samples[100], key)
	}
	delete(history.Samples[499], "http4")
	history.Samples[499]["rsync4"] = 0 // A measured zero is different from missing data.
	raw, _ := json.Marshal(history)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	for _, hours := range []int{1, 6, 24} {
		t.Run(fmt.Sprint(hours), func(t *testing.T) {
			response := c.call("GET", fmt.Sprintf("/api/v1/monitoring?hours=%d", hours), nil, false)
			status(t, response, 200)
			var result struct {
				Panels     []metricPanel `json:"panels"`
				Timestamps []int64       `json:"timestamps"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			for _, panel := range result.Panels {
				wanted := panel.ID == "ipv4" || panel.ID == "iops" || panel.ID == "throughput"
				if panel.Available != wanted {
					t.Fatalf("panel %s available=%t, want %t", panel.ID, panel.Available, wanted)
				}
				for _, series := range panel.Series {
					if len(series.Values) != len(result.Timestamps) {
						t.Fatal("series no longer aligns with shared timestamps")
					}
					if wanted {
						if series.Values[0] != nil || series.Values[1] != nil {
							t.Fatal("startup gap lost during downsampling")
						}
						if series.Stats.Max != 4 || series.Stats.Mean <= 3.9 {
							t.Fatal("missing samples polluted statistics")
						}
					}
				}
			}
			v4 := result.Panels[0].Series
			if v4[0].Stats.Latest != nil || v4[0].Values[len(v4[0].Values)-1] != nil {
				t.Fatal("missing latest measurement presented as zero or a stale value")
			}
			if latest := v4[1].Stats.Latest; latest == nil || *latest != 0 {
				t.Fatal("valid zero measurement was dropped")
			}
		})
	}
}

func TestMonitoringDoesNotRelabelHTTPSOnlyHistory(t *testing.T) {
	c := fixture(t)
	path := filepath.Join(t.TempDir(), "history.json")
	c.app.cfg.Metrics = path
	now := time.Now()
	history := metricsHistory{UpdatedAt: now.UnixMilli(), Samples: []map[string]float64{
		{"time": float64(now.Add(-time.Minute).UnixMilli()), "https4": 20, "rsync4": 3},
		{"time": float64(now.Add(-20 * time.Second).UnixMilli()), "http4": 3, "rsync4": 3},
		{"time": float64(now.Add(-10 * time.Second).UnixMilli()), "http4": 3, "rsync4": 3},
	}}
	raw, _ := json.Marshal(history)
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	response := c.call("GET", "/api/v1/monitoring?hours=1", nil, false)
	status(t, response, 200)
	var result struct{ Panels []metricPanel }
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	series := result.Panels[0].Series[0]
	if !result.Panels[0].Available || series.ID != "http4" || series.LabelKey != "monitor.series.http" || series.Values[0] != nil || series.Stats.Mean != 3 || series.Stats.Max != 3 {
		t.Fatal("HTTPS-only history was incorrectly presented as complete HTTP/S traffic")
	}
	for _, panel := range result.Panels[:2] {
		if panel.Stacked {
			t.Fatalf("%s must plot each protocol's actual rate independently", panel.ID)
		}
	}
	if result.Panels[0].Maximum != 5 {
		t.Fatal("network axis must use the largest individual rate, not their sum")
	}
}
