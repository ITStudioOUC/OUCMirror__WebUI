package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type ConfigNumbers struct {
	Concurrent int `json:"concurrent"`
	Interval   int `json:"interval"`
	Retry      int `json:"retry"`
	Timeout    int `json:"timeout"`
}
type ConfigMirror struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Upstream string `json:"upstream"`
	Interval int    `json:"interval"`
	Retry    int    `json:"retry"`
	Timeout  int    `json:"timeout"`
	Source   string `json:"source,omitempty"`
}
type configFile struct {
	raw string
	doc map[string]any
}
type loadedConfig struct {
	files      map[string]configFile
	mirrors    []ConfigMirror
	global     ConfigNumbers
	hash       string
	includeDir string
}
type fileChange struct {
	Before *string `json:"before"`
	After  *string `json:"after"`
}
type configDraft struct {
	ID       string                `json:"id"`
	Files    map[string]fileChange `json:"files"`
	BaseHash string                `json:"baseHash"`
	Status   string                `json:"status"`
	Created  int64                 `json:"created"`
	Username string                `json:"username"`
}

func asMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
func asTables(value any) []map[string]any {
	switch v := value.(type) {
	case []map[string]any:
		return v
	case []any:
		out := []map[string]any{}
		for _, entry := range v {
			out = append(out, asMap(entry))
		}
		return out
	}
	return nil
}
func integer(value any, fallback int) int {
	switch n := value.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	}
	return fallback
}
func text(value any) string { s, _ := value.(string); return s }
func (a *App) loadConfig() (loadedConfig, error) {
	c := loadedConfig{files: map[string]configFile{}, mirrors: []ConfigMirror{}}
	if a.cfg.WorkerConfig == "" {
		return c, errors.New("CONFIG_UNAVAILABLE")
	}
	read := func(path string) error {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc map[string]any
		if _, err = toml.Decode(string(raw), &doc); err != nil {
			return err
		}
		c.files[path] = configFile{string(raw), doc}
		return nil
	}
	if err := read(a.cfg.WorkerConfig); err != nil {
		return c, err
	}
	main := c.files[a.cfg.WorkerConfig].doc
	g := asMap(main["global"])
	c.global = ConfigNumbers{integer(g["concurrent"], 1), integer(g["interval"], 1440), integer(g["retry"], 0), integer(g["timeout"], 86400)}
	pattern := text(asMap(main["include"])["include_mirrors"])
	if pattern != "" {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			return c, err
		}
		for _, path := range paths {
			if err = read(path); err != nil {
				return c, err
			}
		}
		if filepath.Base(pattern) == "*.conf" {
			c.includeDir = filepath.Dir(pattern)
		}
	}
	seen := map[string]bool{}
	var signature strings.Builder
	for _, path := range sortedKeys(c.files) {
		f := c.files[path]
		fmt.Fprintf(&signature, "%d:%s%d:%s", len(path), path, len(f.raw), f.raw)
		for _, m := range asTables(f.doc["mirrors"]) {
			name := text(m["name"])
			if !validName(name) || seen[name] {
				return c, errors.New("invalid or duplicate mirror name")
			}
			seen[name] = true
			c.mirrors = append(c.mirrors, ConfigMirror{name, text(m["provider"]), text(m["upstream"]), integer(m["interval"], c.global.Interval), integer(m["retry"], c.global.Retry), integer(m["timeout"], c.global.Timeout), path})
		}
	}
	c.hash = tokenHash(signature.String())
	return c, nil
}
func (a *App) configView(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.require(w, r, true); !ok {
		return
	}
	a.configMu.Lock()
	defer a.configMu.Unlock()
	c, err := a.loadConfig()
	if err != nil {
		a.logger.Printf("read config: %v", err)
		fail(w, 503, "CONFIG_UNAVAILABLE")
		return
	}
	respond(w, 200, map[string]any{"worker": a.cfg.Worker, "baseHash": c.hash, "global": c.global, "mirrors": c.mirrors, "canAdd": c.includeDir != ""})
}
func validNumbers(interval, retry, timeout int) bool {
	return interval >= 1 && interval <= 525600 && retry >= 0 && retry <= 20 && timeout >= 30 && timeout <= 1209600
}
func validUpstream(value string) bool {
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "rsync" || u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil && !strings.ContainsAny(value, "\r\n\t `;$|<>\\\"'") && len(value) < 2048
}
func (a *App) configPreview(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, true)
	if !ok {
		return
	}
	var body struct {
		BaseHash string         `json:"baseHash"`
		Global   *ConfigNumbers `json:"global"`
		Mirror   *ConfigMirror  `json:"mirror"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	a.configMu.Lock()
	defer a.configMu.Unlock()
	c, err := a.loadConfig()
	if err != nil {
		fail(w, 503, "CONFIG_UNAVAILABLE")
		return
	}
	if body.BaseHash != c.hash {
		fail(w, 409, "CONFLICT")
		return
	}
	changed := map[string]bool{}
	if body.Global != nil {
		g := body.Global
		if g.Concurrent < 1 || g.Concurrent > 64 || !validNumbers(g.Interval, g.Retry, g.Timeout) {
			fail(w, 400, "INVALID_INPUT")
			return
		}
		if *g != c.global {
			f := c.files[a.cfg.WorkerConfig]
			m := asMap(f.doc["global"])
			m["concurrent"], m["interval"], m["retry"], m["timeout"] = g.Concurrent, g.Interval, g.Retry, g.Timeout
			f.doc["global"] = m
			changed[a.cfg.WorkerConfig] = true
		}
	}
	if body.Mirror != nil {
		m := body.Mirror
		if !validName(m.Name) || !validUpstream(m.Upstream) || !validNumbers(m.Interval, m.Retry, m.Timeout) {
			fail(w, 400, "INVALID_INPUT")
			return
		}
		var current *ConfigMirror
		for _, entry := range c.mirrors {
			if entry.Name == m.Name {
				copy := entry
				current = &copy
				break
			}
		}
		if current != nil {
			if current.Provider != m.Provider {
				fail(w, 400, "INVALID_INPUT")
				return
			}
			if current.Upstream != m.Upstream || current.Interval != m.Interval || current.Retry != m.Retry || current.Timeout != m.Timeout {
				f := c.files[current.Source]
				for _, table := range asTables(f.doc["mirrors"]) {
					if text(table["name"]) == m.Name {
						table["upstream"], table["interval"], table["retry"], table["timeout"] = m.Upstream, m.Interval, m.Retry, m.Timeout
					}
				}
				changed[current.Source] = true
			}
		} else {
			if c.includeDir == "" || (m.Provider != "rsync" && m.Provider != "two-stage-rsync") {
				fail(w, 400, "INVALID_INPUT")
				return
			}
			path := filepath.Join(c.includeDir, "mirrorouc-"+m.Name+".conf")
			if _, err = os.Lstat(path); !os.IsNotExist(err) {
				fail(w, 409, "CONFLICT")
				return
			}
			entry := map[string]any{"name": m.Name, "provider": m.Provider, "upstream": m.Upstream, "interval": m.Interval, "retry": m.Retry, "timeout": m.Timeout}
			if m.Provider == "two-stage-rsync" {
				entry["stage1_profile"] = "debian"
			}
			c.files[path] = configFile{"", map[string]any{"mirrors": []map[string]any{entry}}}
			changed[path] = true
		}
	}
	if len(changed) == 0 {
		fail(w, 400, "NO_CHANGES")
		return
	}
	files := map[string]fileChange{}
	for path := range changed {
		f := c.files[path]
		var b bytes.Buffer
		if err = toml.NewEncoder(&b).Encode(f.doc); err != nil {
			a.internal(w, err)
			return
		}
		after := b.String()
		var before *string
		if _, err = os.Stat(path); err == nil {
			old := f.raw
			before = &old
		}
		files[path] = fileChange{before, &after}
	}
	id := tokenHash(randomToken())[:24]
	raw, _ := json.Marshal(files)
	now := time.Now().UnixMilli()
	_, err = a.db.Exec("INSERT INTO config_versions(id,files,base_hash,status,created,username) VALUES(?,?,?,'draft',?,?)", id, string(raw), c.hash, now, s.Username)
	if err != nil {
		a.internal(w, err)
		return
	}
	a.audit(s, "config_preview", id, "draft")
	respond(w, 201, configDraft{id, files, c.hash, "draft", now, s.Username})
}
func atomicFile(path string, content *string) error {
	if content == nil {
		return os.Remove(path)
	}
	mode := os.FileMode(0660)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".mirrorouc-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err == nil {
		_, err = f.WriteString(*content)
	}
	if err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func (a *App) readVersion(id string) (configDraft, error) {
	var v configDraft
	var raw string
	v.ID = id
	err := a.db.QueryRow("SELECT files,base_hash,status,created,username FROM config_versions WHERE id=?", id).Scan(&raw, &v.BaseHash, &v.Status, &v.Created, &v.Username)
	if err != nil {
		return v, err
	}
	err = json.Unmarshal([]byte(raw), &v.Files)
	return v, err
}
func (a *App) applyVersion(r *http.Request, v configDraft, s Session) (string, error) {
	c, err := a.loadConfig()
	if err != nil {
		return "CONFIG_UNAVAILABLE", err
	}
	if c.hash != v.BaseHash {
		return "CONFLICT", errConflict
	}
	for path, change := range v.Files {
		raw, e := os.ReadFile(path)
		if change.Before == nil {
			if !os.IsNotExist(e) {
				return "CONFLICT", errConflict
			}
		} else if e != nil || string(raw) != *change.Before {
			return "CONFLICT", errConflict
		}
	}
	if a.cfg.ConfigSocket != "" {
		conn, err := net.DialTimeout("unix", a.cfg.ConfigSocket, 5*time.Second)
		if err != nil {
			return "CONFIG_UNAVAILABLE", err
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(28 * time.Second))
		if err = json.NewEncoder(conn).Encode(v); err != nil {
			return "CONFIG_STATE_UNCERTAIN", err
		}
		var result struct {
			Code string `json:"code"`
		}
		if err = json.NewDecoder(bufio.NewReader(conn)).Decode(&result); err != nil {
			return "CONFIG_STATE_UNCERTAIN", err
		}
		if result.Code != "" {
			a.db.Exec("UPDATE config_versions SET status='failed' WHERE id=?", v.ID)
			return result.Code, errors.New(result.Code)
		}
		if _, err = a.db.Exec("UPDATE config_versions SET status='applied' WHERE id=?", v.ID); err != nil {
			return "CONFIG_STATE_UNCERTAIN", err
		}
		a.audit(s, "config_apply", v.ID, "submitted")
		return "", nil
	}
	written := []string{}
	restore := func() error {
		var result error
		for i := len(written) - 1; i >= 0; i-- {
			path := written[i]
			if e := atomicFile(path, v.Files[path].Before); e != nil {
				result = e
			}
		}
		return result
	}
	for _, path := range sortedKeys(v.Files) {
		if err = atomicFile(path, v.Files[path].After); err != nil {
			if e := restore(); e != nil {
				return "CONFIG_ROLLBACK_FAILED", e
			}
			return "CONFIG_APPLY_FAILED", err
		}
		written = append(written, path)
	}
	if _, err = a.loadConfig(); err == nil {
		err = a.sendCommand(r.Context(), "", "reload")
	}
	if err != nil {
		restoreErr := restore()
		if restoreErr == nil {
			restoreErr = a.sendCommand(r.Context(), "", "reload")
		}
		a.db.Exec("UPDATE config_versions SET status='failed' WHERE id=?", v.ID)
		if restoreErr != nil {
			return "CONFIG_ROLLBACK_FAILED", restoreErr
		}
		return "CONFIG_APPLY_FAILED", err
	}
	if _, err = a.db.Exec("UPDATE config_versions SET status='applied' WHERE id=?", v.ID); err != nil {
		return "CONFIG_STATE_UNCERTAIN", err
	}
	a.audit(s, "config_apply", v.ID, "submitted")
	return "", nil
}
func (a *App) configApply(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, true)
	if !ok {
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	a.configMu.Lock()
	defer a.configMu.Unlock()
	v, err := a.readVersion(body.ID)
	if err != nil {
		fail(w, 404, "NOT_FOUND")
		return
	}
	if v.Status != "draft" {
		fail(w, 409, "CONFLICT")
		return
	}
	code, err := a.applyVersion(r, v, s)
	if err != nil {
		a.logger.Printf("config apply: %v", err)
		a.audit(s, "config_apply", v.ID, code)
		status := 502
		if code == "CONFLICT" {
			status = 409
		}
		fail(w, status, code)
		return
	}
	respond(w, 202, map[string]string{"status": "submitted"})
}
func (a *App) configVersions(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.require(w, r, true); !ok {
		return
	}
	rows, err := a.db.Query("SELECT id,base_hash,status,created,username FROM config_versions ORDER BY created DESC LIMIT 50")
	if err != nil {
		a.internal(w, err)
		return
	}
	defer rows.Close()
	out := []configDraft{}
	for rows.Next() {
		var v configDraft
		if err = rows.Scan(&v.ID, &v.BaseHash, &v.Status, &v.Created, &v.Username); err != nil {
			a.internal(w, err)
			return
		}
		out = append(out, v)
	}
	respond(w, 200, out)
}
func (a *App) configRollback(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, true)
	if !ok {
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	a.configMu.Lock()
	defer a.configMu.Unlock()
	old, err := a.readVersion(body.ID)
	if err != nil {
		fail(w, 404, "NOT_FOUND")
		return
	}
	if old.Status != "applied" {
		fail(w, 409, "CONFLICT")
		return
	}
	c, err := a.loadConfig()
	if err != nil {
		fail(w, 503, "CONFIG_UNAVAILABLE")
		return
	}
	reverse := map[string]fileChange{}
	for path, change := range old.Files {
		raw, e := os.ReadFile(path)
		if change.After == nil {
			if !os.IsNotExist(e) {
				fail(w, 409, "CONFLICT")
				return
			}
		} else if e != nil || string(raw) != *change.After {
			fail(w, 409, "CONFLICT")
			return
		}
		reverse[path] = fileChange{change.After, change.Before}
	}
	id := tokenHash(randomToken())[:24]
	now := time.Now().UnixMilli()
	raw, _ := json.Marshal(reverse)
	v := configDraft{id, reverse, c.hash, "draft", now, s.Username}
	if _, err = a.db.Exec("INSERT INTO config_versions(id,files,base_hash,status,created,username) VALUES(?,?,?,'draft',?,?)", id, string(raw), c.hash, now, s.Username); err != nil {
		a.internal(w, err)
		return
	}
	code, err := a.applyVersion(r, v, s)
	if err != nil {
		a.audit(s, "config_rollback", old.ID, code)
		fail(w, 502, code)
		return
	}
	a.audit(s, "config_rollback", old.ID, "submitted")
	respond(w, 202, map[string]string{"status": "submitted"})
}
