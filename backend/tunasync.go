package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) managerGet(ctx context.Context, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(a.cfg.Manager, "/")+path, nil)
	if err != nil {
		return err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("manager HTTP %d", res.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(target)
}
func (a *App) sendCommand(ctx context.Context, name, action string) error {
	if a.cfg.Worker == "" {
		return errors.New("worker not configured")
	}
	data, _ := json.Marshal(map[string]any{"worker_id": a.cfg.Worker, "mirror_id": name, "cmd": action, "args": []string{}, "options": map[string]bool{}})
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(a.cfg.Manager, "/")+"/cmd", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("manager command HTTP %d", res.StatusCode)
	}
	return nil
}
func (a *App) cachedJobs() ([]Job, int64, string) {
	a.jobsMu.RLock()
	defer a.jobsMu.RUnlock()
	jobs := append([]Job{}, a.jobs...)
	return jobs, a.jobsUpdated, a.jobsError
}
func (a *App) loadCache() {
	var raw string
	var updated int64
	if a.db.QueryRow("SELECT data,updated FROM cache WHERE key='jobs'").Scan(&raw, &updated) == nil {
		var jobs []Job
		if json.Unmarshal([]byte(raw), &jobs) == nil {
			a.jobs = jobs
			a.jobsUpdated = updated
			a.jobsError = "TUNASYNC_UNAVAILABLE"
		}
	}
}
func (a *App) refresh(ctx context.Context) {
	jobs := []Job{}
	err := error(nil)
	if a.cfg.Worker == "" {
		err = errors.New("worker not configured")
	} else {
		err = a.managerGet(ctx, "/workers/"+url.PathEscape(a.cfg.Worker)+"/jobs", &jobs)
	}
	a.jobsMu.Lock()
	if err != nil {
		a.jobsError = "TUNASYNC_UNAVAILABLE"
		a.jobsMu.Unlock()
		a.logger.Printf("tunasync poll: %v", err)
		return
	}
	a.jobs = jobs
	a.jobsUpdated = time.Now().UnixMilli()
	a.jobsError = ""
	updated := a.jobsUpdated
	a.jobsMu.Unlock()
	raw, _ := json.Marshal(jobs)
	if _, err = a.db.Exec("INSERT INTO cache(key,data,updated) VALUES('jobs',?,?) ON CONFLICT(key) DO UPDATE SET data=excluded.data,updated=excluded.updated", string(raw), updated); err != nil {
		a.logger.Printf("cache: %v", err)
	}
}
func (a *App) poll(ctx context.Context) {
	a.refresh(ctx)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refresh(ctx)
		}
	}
}
func stamp(value time.Time) *int64 {
	if value.IsZero() || value.Unix() <= 0 {
		return nil
	}
	ms := value.UnixMilli()
	return &ms
}

var humanSize = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*([kmgtpe]?)(?:i?b)?\s*$`)

func parseSize(value string) *float64 {
	parts := humanSize.FindStringSubmatch(value)
	if parts == nil {
		return nil
	}
	n, err := strconv.ParseFloat(parts[1], 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return nil
	}
	power := strings.Index("kmgtpe", strings.ToLower(parts[2])) + 1
	if parts[2] == "" {
		power = 0
	}
	n *= math.Pow(1024, float64(power))
	return &n
}
func redactURL(value string) string {
	u, err := url.Parse(value)
	if err != nil {
		return ""
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
func publicMirror(job Job, meta MirrorMetadata) PublicMirror {
	status := map[string]string{"success": "success", "syncing": "syncing", "pre-syncing": "waiting", "failed": "failed", "paused": "paused", "disabled": "paused", "none": "unknown"}[job.Status]
	if status == "" {
		status = "unknown"
	}
	return PublicMirror{Name: job.Name, Description: meta.Description, Help: meta.Help, Category: meta.Category, Status: status, LastSuccess: stamp(job.LastUpdate), LastAttempt: stamp(job.LastStarted), SyncStarted: stamp(job.LastStarted), NextSync: stamp(job.NextSchedule), Size: parseSize(job.Size), Website: meta.Website, Upstream: redactURL(job.Upstream), BrowseURL: "/mirrors/" + url.PathEscape(job.Name) + "/"}
}
func (a *App) command(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, true)
	if !ok {
		return
	}
	name, action := r.PathValue("name"), r.PathValue("action")
	if action != "start" && action != "stop" && action != "restart" && action != "disable" {
		fail(w, 400, "INVALID_ACTION")
		return
	}
	jobs, _, code := a.cachedJobs()
	found := false
	for _, job := range jobs {
		if job.Name == name {
			found = true
			break
		}
	}
	if !validName(name) || !found {
		fail(w, 404, "NOT_FOUND")
		return
	}
	if code != "" {
		fail(w, 503, "TUNASYNC_UNAVAILABLE")
		return
	}
	if err := a.sendCommand(r.Context(), name, action); err != nil {
		a.audit(s, "job_"+action, name, "uncertain")
		a.logger.Printf("command: %v", err)
		fail(w, 502, "COMMAND_UNCERTAIN")
		return
	}
	a.audit(s, "job_"+action, name, "submitted")
	a.refresh(r.Context())
	respond(w, 202, map[string]string{"status": "submitted"})
}
func (a *App) logs(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.require(w, r, true); !ok {
		return
	}
	name := r.PathValue("name")
	if a.cfg.Logs == "" {
		fail(w, 503, "LOGS_UNAVAILABLE")
		return
	}
	if !validName(name) {
		fail(w, 400, "INVALID_INPUT")
		return
	}
	base, baseErr := filepath.EvalSymlinks(a.cfg.Logs)
	if baseErr != nil {
		fail(w, 503, "LOGS_UNAVAILABLE")
		return
	}
	root, err := filepath.EvalSymlinks(filepath.Join(base, name))
	if err != nil {
		fail(w, 404, "NOT_FOUND")
		return
	}
	rootRel, err := filepath.Rel(base, root)
	if err != nil || rootRel == ".." || strings.HasPrefix(rootRel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rootRel) {
		fail(w, 403, "FORBIDDEN")
		return
	}
	path, err := filepath.EvalSymlinks(filepath.Join(root, "latest"))
	if err != nil {
		fail(w, 404, "NOT_FOUND")
		return
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		fail(w, 403, "FORBIDDEN")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		fail(w, 404, "NOT_FOUND")
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		fail(w, 400, "INVALID_INPUT")
		return
	}
	truncated := info.Size() > 65536
	if truncated {
		if _, err = f.Seek(-65536, io.SeekEnd); err != nil {
			a.internal(w, err)
			return
		}
	}
	content, err := io.ReadAll(io.LimitReader(f, 65536))
	if err != nil {
		a.internal(w, err)
		return
	}
	respond(w, 200, map[string]any{"content": strings.ToValidUTF8(string(content), "�"), "truncated": truncated})
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
