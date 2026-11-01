package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testClient struct {
	t      *testing.T
	app    *App
	cookie *http.Cookie
	csrf   string
}

func fixture(t *testing.T) *testClient {
	t.Helper()
	root := t.TempDir()
	cfgDir := filepath.Join(root, "config")
	os.MkdirAll(filepath.Join(cfgDir, "mirrors.d"), 0755)
	worker := filepath.Join(cfgDir, "worker.conf")
	inc := filepath.Join(cfgDir, "mirrors.d", "source.conf")
	global := "[global]\nname='test'\nconcurrent=2\ninterval=60\nretry=2\ntimeout=600\n[include]\ninclude_mirrors=" + strconvQuote(filepath.ToSlash(filepath.Join(cfgDir, "mirrors.d", "*.conf"))) + "\n"
	os.WriteFile(worker, []byte(global), 0644)
	os.WriteFile(inc, []byte("[[mirrors]]\nname='alpine'\nprovider='rsync'\ncommand='/trusted/parallel-rsync'\nupstream='rsync://example.invalid/alpine/'\ninterval=60\nrsync_options=['--no-owner']\n"), 0644)
	manager := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workers/test/jobs":
			respond(w, 200, []Job{{Name: "alpine", Worker: "test", Status: "failed", LastUpdate: time.Now().Add(-time.Hour), LastStarted: time.Now().Add(-time.Minute), Upstream: "https://user:secret@example.invalid/repo?token=secret", Size: "1.5G"}})
		case "/workers":
			respond(w, 200, []Worker{{ID: "test"}})
		case "/cmd":
			respond(w, 200, map[string]bool{"ok": true})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(manager.Close)
	a, err := newApp(Config{Data: filepath.Join(root, "data"), Static: root, Manager: manager.URL, Worker: "test", WorkerConfig: worker, Logs: filepath.Join(root, "logs")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.db.Close() })
	a.logger.SetOutput(io.Discard)
	hash, err := hashPassword("test-only-password-123")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.db.Exec("INSERT INTO users(username,password,role,created) VALUES('admin',?,'admin',0),('editor',?,'editor',0)", hash, hash)
	if err != nil {
		t.Fatal(err)
	}
	a.refresh(context.Background())
	return &testClient{t: t, app: a}
}
func strconvQuote(s string) string { b, _ := json.Marshal(s); return string(b) }
func (c *testClient) call(method, path string, body any, csrf bool) *httptest.ResponseRecorder {
	c.t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	r := httptest.NewRequest(method, "http://localhost"+path, bytes.NewReader(raw))
	r.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	if csrf {
		r.Header.Set("X-CSRF-Token", c.csrf)
	}
	w := httptest.NewRecorder()
	c.app.handler().ServeHTTP(w, r)
	return w
}
func (c *testClient) login(user string) {
	c.t.Helper()
	w := c.call("POST", "/api/v1/auth/login", map[string]string{"username": user, "password": "test-only-password-123"}, false)
	if w.Code != 200 {
		c.t.Fatalf("login %d: %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		c.t.Fatal("unsafe session cookie")
	}
	c.cookie = cookies[0]
	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	c.csrf = result["csrf"]
}
func status(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("want %d got %d: %s", want, w.Code, w.Body.String())
	}
}
func TestAuthenticationAndRoles(t *testing.T) {
	c := fixture(t)
	status(t, c.call("GET", "/api/v1/admin/overview", nil, false), 401)
	c.login("editor")
	status(t, c.call("GET", "/api/v1/admin/config", nil, false), 403)
	ann := Announcement{Title: "测试", Body: "草稿", Date: time.Now().UTC().Format(time.RFC3339)}
	status(t, c.call("PUT", "/api/v1/admin/records/announcements/test", map[string]any{"data": ann, "revision": 0}, false), 403)
	status(t, c.call("PUT", "/api/v1/admin/records/announcements/test", map[string]any{"data": ann, "revision": 0}, true), 200)
	w := c.call("GET", "/api/v1/public", nil, false)
	status(t, w, 200)
	var public struct {
		Announcements []Record       `json:"announcements"`
		Mirrors       []PublicMirror `json:"mirrors"`
	}
	json.Unmarshal(w.Body.Bytes(), &public)
	if len(public.Announcements) != 0 || len(public.Mirrors) != 1 || strings.Contains(public.Mirrors[0].Upstream, "secret") || strings.Contains(public.Mirrors[0].Upstream, "user") {
		t.Fatal("draft or private upstream data exposed")
	}
	ann.Published = true
	status(t, c.call("PUT", "/api/v1/admin/records/announcements/test", map[string]any{"data": ann, "revision": 1}, true), 200)
	status(t, c.call("PUT", "/api/v1/admin/records/announcements/test", map[string]any{"data": ann, "revision": 1}, true), 409)
	status(t, c.call("PUT", "/api/v1/auth/password", map[string]string{"current": "test-only-password-123", "password": "new-test-only-password-456"}, true), 200)
	status(t, c.call("GET", "/api/v1/auth/session", nil, false), 401)
}
func TestCrossOriginAndManagerActions(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	r := httptest.NewRequest("POST", "http://localhost/api/v1/admin/jobs/alpine/start", nil)
	r.AddCookie(c.cookie)
	r.Header.Set("Origin", "https://attacker.invalid")
	r.Header.Set("X-CSRF-Token", c.csrf)
	w := httptest.NewRecorder()
	c.app.handler().ServeHTTP(w, r)
	status(t, w, 403)
	status(t, c.call("POST", "/api/v1/admin/jobs/alpine/not-a-command", nil, true), 400)
	status(t, c.call("POST", "/api/v1/admin/jobs/missing/start", nil, true), 404)
	status(t, c.call("POST", "/api/v1/admin/jobs/alpine/start", nil, true), 202)
}
func TestConfigApplyRollbackAndConflict(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	initial, err := c.app.loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	mirror := initial.mirrors[0]
	mirror.Interval = 120
	w := c.call("POST", "/api/v1/admin/config/preview", map[string]any{"baseHash": initial.hash, "mirror": mirror}, true)
	status(t, w, 201)
	var draft configDraft
	json.Unmarshal(w.Body.Bytes(), &draft)
	status(t, c.call("POST", "/api/v1/admin/config/apply", map[string]string{"id": draft.ID}, true), 202)
	updated, err := c.app.loadConfig()
	if err != nil || updated.mirrors[0].Interval != 120 {
		t.Fatal("config was not applied")
	}
	for _, f := range updated.files {
		if strings.Contains(f.raw, "[[mirrors]]") && (!strings.Contains(f.raw, "/trusted/parallel-rsync") || !strings.Contains(f.raw, "--no-owner")) {
			t.Fatal("custom configuration was lost")
		}
	}
	status(t, c.call("POST", "/api/v1/admin/config/rollback", map[string]string{"id": draft.ID}, true), 202)
	restored, err := c.app.loadConfig()
	if err != nil || restored.hash != initial.hash {
		t.Fatal("rollback did not restore exact bytes")
	}
	w = c.call("POST", "/api/v1/admin/config/preview", map[string]any{"baseHash": restored.hash, "mirror": mirror}, true)
	status(t, w, 201)
	json.Unmarshal(w.Body.Bytes(), &draft)
	f, err := os.OpenFile(c.app.cfg.WorkerConfig, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("\n# concurrent edit\n")
	f.Close()
	status(t, c.call("POST", "/api/v1/admin/config/apply", map[string]string{"id": draft.ID}, true), 409)
}
func TestConfigReloadFailureRestoresFiles(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	initial, _ := c.app.loadConfig()
	mirror := initial.mirrors[0]
	mirror.Interval = 90
	w := c.call("POST", "/api/v1/admin/config/preview", map[string]any{"baseHash": initial.hash, "mirror": mirror}, true)
	status(t, w, 201)
	var draft configDraft
	json.Unmarshal(w.Body.Bytes(), &draft)
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unavailable", 503) }))
	defer broken.Close()
	c.app.cfg.Manager = broken.URL
	status(t, c.call("POST", "/api/v1/admin/config/apply", map[string]string{"id": draft.ID}, true), 502)
	restored, err := c.app.loadConfig()
	if err != nil || restored.hash != initial.hash {
		t.Fatal("failed reload did not restore configuration files")
	}
}
func TestNullTimesSizesAndStaleCache(t *testing.T) {
	if stamp(time.Time{}) != nil || parseSize("unknown") != nil || parseSize("NaN") != nil {
		t.Fatal("missing values must remain missing")
	}
	if value := parseSize("1.5G"); value == nil || *value != 1.5*1024*1024*1024 {
		t.Fatal("size conversion")
	}
	c := fixture(t)
	c.app.cfg.Manager = "http://127.0.0.1:1"
	c.app.refresh(context.Background())
	jobs, updated, code := c.app.cachedJobs()
	if len(jobs) != 1 || updated == 0 || code == "" {
		t.Fatal("last valid jobs were lost after upstream failure")
	}
}
