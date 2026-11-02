package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUploadsAreValidatedAndNotListed(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	upload := func(value []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, _ := form.CreateFormFile("file", "../../unsafe.html")
		part.Write(value)
		form.Close()
		request := httptest.NewRequest("POST", "http://localhost/api/v1/admin/uploads", &body)
		request.Header.Set("Content-Type", form.FormDataContentType())
		request.Header.Set("X-CSRF-Token", c.csrf)
		request.AddCookie(c.cookie)
		response := httptest.NewRecorder()
		c.app.handler().ServeHTTP(response, request)
		return response
	}
	status(t, upload([]byte("<script>unsafe()</script>")), 400)
	var pixel bytes.Buffer
	png.Encode(&pixel, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	result := upload(pixel.Bytes())
	status(t, result, 201)
	var value map[string]string
	json.Unmarshal(result.Body.Bytes(), &value)
	status(t, c.call("GET", value["url"], nil, false), 200)
	status(t, c.call("GET", "/uploads/", nil, false), 404)
	status(t, c.call("GET", "/uploads/mirrorouc.db", nil, false), 404)
}

func TestMonitoringUsesRawStatistics(t *testing.T) {
	c := fixture(t)
	history := metricsHistory{UpdatedAt: time.Now().UnixMilli(), MemoryTotal: 32, Samples: []map[string]float64{}}
	for index := 0; index < 500; index++ {
		value := float64(1)
		if index == 1 {
			value = 200
		}
		history.Samples = append(history.Samples, map[string]float64{"time": float64(time.Now().Add(time.Duration(index-500) * time.Second).UnixMilli()), "http4": value, "rsync4": 0})
	}
	raw, _ := json.Marshal(history)
	path := filepath.Join(t.TempDir(), "history.json")
	os.WriteFile(path, raw, 0644)
	c.app.cfg.Metrics = path
	response := c.call("GET", "/api/v1/monitoring?hours=1", nil, false)
	status(t, response, 200)
	var result struct {
		Panels     []metricPanel
		Timestamps []int64
	}
	json.Unmarshal(response.Body.Bytes(), &result)
	if len(result.Timestamps) != 241 || result.Panels[0].Series[0].Stats.Max != 200 || !result.Panels[0].Available || result.Panels[1].Available {
		t.Fatal("incorrect sampled statistics or missing-metric handling")
	}
	status(t, c.call("GET", "/api/v1/monitoring?hours=2", nil, false), 400)
}
func TestJobSpeedsFilterInactiveAndStaleData(t *testing.T) {
	c := fixture(t)
	root := t.TempDir()
	c.app.cfg.Metrics = filepath.Join(root, "history.json")
	c.app.jobsMu.Lock()
	c.app.jobs = []Job{{Name: "debian", Status: "syncing"}, {Name: "ubuntu", Status: "success"}}
	c.app.jobsMu.Unlock()
	snapshot := speedSnapshot{UpdatedAt: time.Now().UnixMilli(), Interval: 2, Available: true, Jobs: map[string]jobSpeed{"debian": {BytesPerSecond: 12345, Connections: 3}, "ubuntu": {BytesPerSecond: 99999}, "secret-job": {BytesPerSecond: 99999}}}
	write := func() {
		raw, _ := json.Marshal(snapshot)
		os.WriteFile(filepath.Join(root, "job-speeds.json"), raw, 0644)
	}
	write()
	response := c.call("GET", "/api/v1/job-speeds", nil, false)
	status(t, response, 200)
	var result speedSnapshot
	json.Unmarshal(response.Body.Bytes(), &result)
	if len(result.Jobs) != 1 || result.Jobs["debian"].BytesPerSecond != 12345 {
		t.Fatal("inactive or unknown jobs leaked into speeds")
	}
	snapshot.UpdatedAt = time.Now().Add(-time.Minute).UnixMilli()
	write()
	response = c.call("GET", "/api/v1/job-speeds", nil, false)
	json.Unmarshal(response.Body.Bytes(), &result)
	// Decode into a new map: encoding/json merges map entries when reusing one.
	result = speedSnapshot{}
	json.Unmarshal(response.Body.Bytes(), &result)
	if !result.Stale || len(result.Jobs) != 0 {
		t.Fatal("outdated rates presented as current")
	}
}
