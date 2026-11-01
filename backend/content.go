package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.+-]{0,127}$`)

func validName(value string) bool {
	return identifier.MatchString(value) && value != "." && value != ".."
}
func validText(value Localized, required bool, limit int) bool {
	has := false
	if len(value) > 20 {
		return false
	}
	for locale, text := range value {
		if len(locale) > 48 || !localeIdentifier.MatchString(locale) {
			return false
		}
		if len(text) > limit {
			return false
		}
		if strings.TrimSpace(text) != "" {
			has = true
		}
	}
	return !required || has
}

var localeIdentifier = regexp.MustCompile(`^[a-zA-Z]{2,3}(?:-[a-zA-Z0-9]{2,8})*$`)

func webURL(value string, empty bool) bool {
	if empty && value == "" {
		return true
	}
	u, e := url.Parse(value)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && len(value) < 2048
}
func navigationURL(value string) bool {
	if webURL(value, false) {
		return true
	}
	if !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || len(value) >= 2048 || strings.ContainsAny(value, "\\\r\n\t") {
		return false
	}
	u, err := url.Parse(value)
	return err == nil && u.Scheme == "" && u.Host == "" && u.User == nil
}
func imageURL(value string) bool {
	if strings.HasPrefix(value, "/assets/") || strings.HasPrefix(value, "/uploads/") {
		return !strings.Contains(value, "..") && !strings.ContainsAny(value, "\\?#")
	}
	u, e := url.Parse(value)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil
}
func strict(raw []byte, value any) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(value) == nil
}
func (a *App) listRecords(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	if kind != "announcements" && kind != "mirrors" && kind != "settings" {
		fail(w, 404, "NOT_FOUND")
		return
	}
	if _, ok := a.require(w, r, kind != "announcements"); !ok {
		return
	}
	records, err := a.records(kind)
	if err != nil {
		a.internal(w, err)
		return
	}
	respond(w, 200, records)
}
func (a *App) putRecord(w http.ResponseWriter, r *http.Request) {
	kind, id := r.PathValue("kind"), r.PathValue("id")
	if !validName(id) || (kind != "announcements" && kind != "mirrors" && kind != "settings") {
		fail(w, 400, "INVALID_INPUT")
		return
	}
	s, ok := a.require(w, r, kind != "announcements")
	if !ok {
		return
	}
	var body struct {
		Data     json.RawMessage `json:"data"`
		Revision int             `json:"revision"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	var value any
	switch kind {
	case "announcements":
		var d Announcement
		if !strict(body.Data, &d) || strings.TrimSpace(d.Title) == "" || len(d.Title) > 300 || len(d.Body) > 250000 || (d.Published && strings.TrimSpace(d.Body) == "") {
			fail(w, 400, "INVALID_INPUT")
			return
		}
		if d.Date == "" {
			d.Date = time.Now().UTC().Format(time.RFC3339)
		}
		if _, err := time.Parse(time.RFC3339, d.Date); err != nil {
			fail(w, 400, "INVALID_INPUT")
			return
		}
		value = d
	case "mirrors":
		var d MirrorMetadata
		if !strict(body.Data, &d) || d.Name != id || !validText(d.Description, false, 1000) || len(d.Help) > 250000 || !webURL(d.Website, true) || (d.Category != "os" && d.Category != "language" && d.Category != "other") {
			fail(w, 400, "INVALID_INPUT")
			return
		}
		value = d
	case "settings":
		var d Settings
		if id != "site" || !strict(body.Data, &d) || !validText(d.Name, true, 200) || !validText(d.Description, false, 2000) || !validText(d.Address, false, 2000) || !imageURL(d.LogoLight) || !imageURL(d.LogoDark) || !imageURL(d.Favicon) || !imageURL(d.Avatar) || len(d.Links) > 50 || len(d.RelatedLinks) > 50 || len(d.Downloads) > 100 {
			fail(w, 400, "INVALID_INPUT")
			return
		}
		var fields map[string]json.RawMessage
		json.Unmarshal(body.Data, &fields)
		if _, provided := fields["relatedLinks"]; !provided {
			// Preserve saved sidebar changes when an older open form submits without this field.
			current, err := a.record("settings", "site")
			var existing Settings
			if err != nil || decodeData(current.Data, &existing) != nil {
				fail(w, 409, "CONFLICT")
				return
			}
			d.RelatedLinks = existing.RelatedLinks
		}
		if d.RelatedLinks == nil {
			d.RelatedLinks = []RelatedLink{}
		}
		for _, link := range d.RelatedLinks {
			if !validText(link.Title, true, 200) || (link.Kind != "help" && link.Kind != "url") || (link.Kind == "help" && link.URL != "") || (link.Kind == "url" && !navigationURL(link.URL)) {
				fail(w, 400, "INVALID_INPUT")
				return
			}
		}
		if d.Email != "" {
			if parsed, err := mail.ParseAddress(d.Email); err != nil || parsed.Address != d.Email {
				fail(w, 400, "INVALID_INPUT")
				return
			}
		}
		for _, link := range d.Links {
			if !validText(link.Title, true, 200) || !webURL(link.URL, false) {
				fail(w, 400, "INVALID_INPUT")
				return
			}
		}
		for _, download := range d.Downloads {
			if strings.TrimSpace(download.Name) == "" || len(download.Name) > 100 || len(download.Version) > 100 || !webURL(download.URL, false) {
				fail(w, 400, "INVALID_INPUT")
				return
			}
		}
		value = d
	}
	result, err := a.saveRecord(kind, id, value, body.Revision)
	if errors.Is(err, errConflict) {
		fail(w, 409, "CONFLICT")
		return
	}
	if err != nil {
		a.internal(w, err)
		return
	}
	a.audit(s, "record_save", kind+"/"+id, "success")
	respond(w, 200, result)
}
func (a *App) deleteRecord(w http.ResponseWriter, r *http.Request) {
	kind, id := r.PathValue("kind"), r.PathValue("id")
	if kind != "announcements" && kind != "mirrors" {
		fail(w, 400, "INVALID_INPUT")
		return
	}
	s, ok := a.require(w, r, kind != "announcements")
	if !ok {
		return
	}
	var body struct {
		Revision int `json:"revision"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	result, err := a.db.Exec("UPDATE records SET deleted=1,revision=revision+1,updated=? WHERE kind=? AND id=? AND revision=? AND deleted=0", time.Now().UnixMilli(), kind, id, body.Revision)
	if err != nil {
		a.internal(w, err)
		return
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		fail(w, 409, "CONFLICT")
		return
	}
	a.audit(s, "record_delete", kind+"/"+id, "success")
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) upload(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, true)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		fail(w, 400, "INVALID_IMAGE")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		fail(w, 400, "INVALID_IMAGE")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, 4<<20+1))
	if err != nil || len(content) > 4<<20 {
		fail(w, 400, "INVALID_IMAGE")
		return
	}
	conf, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || (format != "png" && format != "jpeg") || conf.Width < 1 || conf.Height < 1 || conf.Width > 8192 || conf.Height > 8192 || int64(conf.Width)*int64(conf.Height) > 32000000 {
		fail(w, 400, "INVALID_IMAGE")
		return
	}
	ext := ".png"
	if format == "jpeg" {
		ext = ".jpg"
	}
	name := tokenHash(randomToken())[:24] + ext
	if err = os.WriteFile(filepath.Join(a.cfg.Data, "uploads", name), content, 0644); err != nil {
		a.internal(w, err)
		return
	}
	a.audit(s, "upload", name, "success")
	respond(w, 201, map[string]string{"url": "/uploads/" + name})
}
func (a *App) public(w http.ResponseWriter, r *http.Request) {
	site, err := a.record("settings", "site")
	if err != nil {
		a.internal(w, err)
		return
	}
	metadata, err := a.records("mirrors")
	if err != nil {
		a.internal(w, err)
		return
	}
	metas := map[string]MirrorMetadata{}
	for _, record := range metadata {
		var m MirrorMetadata
		if decodeData(record.Data, &m) == nil {
			metas[record.ID] = m
		}
	}
	jobs, updated, upstreamError := a.cachedJobs()
	mirrors := []PublicMirror{}
	for _, job := range jobs {
		meta, exists := metas[job.Name]
		if exists && !meta.Visible {
			continue
		}
		if !exists {
			meta = MirrorMetadata{Name: job.Name, Category: "other", Visible: true, Description: Localized{}}
		}
		mirrors = append(mirrors, publicMirror(job, meta))
	}
	sort.Slice(mirrors, func(i, j int) bool { return mirrors[i].Name < mirrors[j].Name })
	published, err := a.publishedAnnouncements()
	if err != nil {
		a.internal(w, err)
		return
	}
	// The home page only needs recent links. The archive endpoint returns all bodies.
	if len(published) > 5 {
		published = published[:5]
	}
	respond(w, 200, map[string]any{"site": site.Data, "mirrors": mirrors, "announcements": published, "source": map[string]any{"updatedAt": updated, "stale": upstreamError != "" || updated == 0 || time.Now().UnixMilli()-updated > 60000, "code": upstreamError}})
}

func (a *App) publishedAnnouncements() ([]Record, error) {
	announcements, err := a.records("announcements")
	if err != nil {
		return nil, err
	}
	published := []Record{}
	for _, record := range announcements {
		var d Announcement
		if decodeData(record.Data, &d) == nil && d.Published {
			record.Data = d
			published = append(published, record)
		}
	}
	sort.SliceStable(published, func(i, j int) bool {
		a, b := published[i].Data.(Announcement), published[j].Data.(Announcement)
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		at, _ := time.Parse(time.RFC3339, a.Date)
		bt, _ := time.Parse(time.RFC3339, b.Date)
		if !at.Equal(bt) {
			return at.After(bt)
		}
		return published[i].ID < published[j].ID
	})
	return published, nil
}

func (a *App) announcements(w http.ResponseWriter, r *http.Request) {
	published, err := a.publishedAnnouncements()
	if err != nil {
		a.internal(w, err)
		return
	}
	respond(w, 200, published)
}
func (a *App) overview(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.require(w, r, false); !ok {
		return
	}
	jobs, updated, code := a.cachedJobs()
	workers := []Worker{}
	err := a.managerGet(r.Context(), "/workers", &workers)
	if err != nil && code == "" {
		code = "TUNASYNC_UNAVAILABLE"
	}
	respond(w, 200, map[string]any{"jobs": jobs, "workers": workers, "worker": a.cfg.Worker, "updatedAt": updated, "code": code, "configAvailable": a.cfg.WorkerConfig != "", "logsAvailable": a.cfg.Logs != ""})
}
func (a *App) auditList(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.require(w, r, true); !ok {
		return
	}
	page, pageSize := 1, 10
	paginated := r.URL.Query().Has("page")
	if paginated {
		var err error
		page, err = strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page < 1 {
			fail(w, 400, "INVALID_INPUT")
			return
		}
	}
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		a.internal(w, err)
		return
	}
	defer tx.Rollback()
	total, totalPages, limit, offset := 0, 1, 200, 0
	if paginated {
		if err := tx.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM audit").Scan(&total); err != nil {
			a.internal(w, err)
			return
		}
		totalPages = max(1, (total+pageSize-1)/pageSize)
		page = min(page, totalPages)
		limit, offset = pageSize, (page-1)*pageSize
	}
	rows, err := tx.QueryContext(r.Context(), "SELECT id,username,action,target,result,created FROM audit ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		a.internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, created int64
		var username, action, target, result string
		if err = rows.Scan(&id, &username, &action, &target, &result, &created); err != nil {
			a.internal(w, err)
			return
		}
		out = append(out, map[string]any{"id": id, "username": username, "action": action, "target": target, "result": result, "created": created})
	}
	if err := rows.Err(); err != nil {
		a.internal(w, err)
		return
	}
	if err := rows.Close(); err != nil {
		a.internal(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		a.internal(w, err)
		return
	}
	if paginated {
		respond(w, 200, map[string]any{"items": out, "page": page, "pageSize": pageSize, "total": total, "totalPages": totalPages})
		return
	}
	// Keep the array response for already-open admin pages from older releases.
	respond(w, 200, out)
}
func (a *App) users(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.require(w, r, true); !ok {
		return
	}
	rows, err := a.db.Query("SELECT id,username,role,created FROM users ORDER BY id")
	if err != nil {
		a.internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, created int64
		var username, role string
		if err = rows.Scan(&id, &username, &role, &created); err != nil {
			a.internal(w, err)
			return
		}
		out = append(out, map[string]any{"id": id, "username": username, "role": role, "created": created})
	}
	respond(w, 200, out)
}
func (a *App) createUser(w http.ResponseWriter, r *http.Request) {
	s, ok := a.require(w, r, true)
	if !ok {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if !validName(body.Username) || (body.Role != "admin" && body.Role != "editor") {
		fail(w, 400, "INVALID_INPUT")
		return
	}
	hash, err := hashPassword(body.Password)
	if err != nil {
		fail(w, 400, "INVALID_PASSWORD")
		return
	}
	if _, err = a.db.Exec("INSERT INTO users(username,password,role,created) VALUES(?,?,?,?)", body.Username, hash, body.Role, time.Now().UnixMilli()); err != nil {
		fail(w, 409, "CONFLICT")
		return
	}
	a.audit(s, "user_create", body.Username, "success")
	respond(w, 201, map[string]bool{"ok": true})
}
