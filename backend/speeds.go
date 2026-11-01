package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type jobSpeed struct {
	BytesPerSecond float64 `json:"bytesPerSecond"`
	Connections    int     `json:"connections"`
}
type speedSnapshot struct {
	UpdatedAt int64               `json:"updatedAt"`
	Interval  float64             `json:"interval"`
	Available bool                `json:"available"`
	Jobs      map[string]jobSpeed `json:"jobs"`
	Stale     bool                `json:"stale"`
}

func (a *App) jobSpeeds(w http.ResponseWriter, r *http.Request) {
	if a.cfg.Metrics == "" {
		fail(w, 503, "SPEEDS_UNAVAILABLE")
		return
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(a.cfg.Metrics), "job-speeds.json"))
	var snapshot speedSnapshot
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &snapshot) != nil {
		fail(w, 503, "SPEEDS_UNAVAILABLE")
		return
	}
	snapshot.Stale = time.Now().UnixMilli()-snapshot.UpdatedAt > 15000 || snapshot.UpdatedAt > time.Now().Add(time.Minute).UnixMilli()
	filtered := map[string]jobSpeed{}
	jobs, _, code := a.cachedJobs()
	if code == "" && snapshot.Available && !snapshot.Stale {
		for _, job := range jobs {
			if speed, exists := snapshot.Jobs[job.Name]; exists && job.Status == "syncing" && speed.BytesPerSecond >= 0 && speed.Connections >= 0 {
				filtered[job.Name] = speed
			}
		}
	}
	snapshot.Jobs = filtered
	respond(w, 200, snapshot)
}
