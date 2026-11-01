package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDownloadCatalogLegacyCompatibilityAndPersistence(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	initial, _ := c.app.record("settings", "site")
	var old map[string]any
	decodeData(initial.Data, &old)
	old["downloads"] = []map[string]string{
		{"name": "Ubuntu", "version": "24.04 (amd64)", "url": "https://example.invalid/ubuntu.iso"},
		{"name": "Ubuntu", "version": "24.04 (arm64)", "url": "https://example.invalid/ubuntu-arm.iso"},
		{"name": "Tool", "version": "", "url": "https://example.invalid/tool.zip"},
	}
	legacy, err := c.app.saveRecord("settings", "site", old, initial.Revision)
	if err != nil {
		t.Fatal(err)
	}
	var rawBefore, rawAfter string
	c.app.db.QueryRow("SELECT data FROM records WHERE kind='settings' AND id='site'").Scan(&rawBefore)
	read := c.call("GET", "/api/v1/public", nil, false)
	status(t, read, 200)
	var public struct{ Site Settings }
	if err := json.Unmarshal(read.Body.Bytes(), &public); err != nil {
		t.Fatal(err)
	}
	items := public.Site.Downloads
	if len(items) != 2 || items[0].Name != "Ubuntu" || len(items[0].Files) != 2 || items[0].Files[1].Label != "24.04 (arm64)" || items[1].Files[0].Label != "Tool" || items[0].Category != "software" || !validName(items[0].ID) || items[0].URL != "" {
		t.Fatalf("legacy projects or version links lost: %+v", items)
	}
	c.app.db.QueryRow("SELECT data FROM records WHERE kind='settings' AND id='site'").Scan(&rawAfter)
	if rawBefore != rawAfter {
		t.Fatal("reading public data changed stored settings")
	}
	// Old open forms may still submit the former flat download structure.
	w := c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": old, "revision": legacy.Revision}, true)
	status(t, w, 200)
	var saved Record
	json.Unmarshal(w.Body.Bytes(), &saved)
	var settings Settings
	decodeData(saved.Data, &settings)
	if !reflect.DeepEqual(settings.Downloads, items) {
		t.Fatal("older form lost downloads or changed stable project IDs")
	}
	settings.Downloads[0].Category = "os"
	settings.Downloads[0].Description = "## Ubuntu\n\n选择适合设备架构的版本。"
	settings.Downloads[0].Files[0].URL = "/mirrors/ubuntu-releases/24.04/ubuntu.iso"
	settings.Downloads = append(settings.Downloads, Download{ID: "fonts", Name: "字体", Category: "fonts", Files: []DownloadFile{{Label: "字体压缩包", URL: "https://example.invalid/font.zip"}}})
	w = c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": saved.Revision}, true)
	status(t, w, 200)
	status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": saved.Revision}, true), 409)
	json.Unmarshal(w.Body.Bytes(), &saved)
	var omitted map[string]any
	decodeData(settings, &omitted)
	delete(omitted, "downloads")
	w = c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": omitted, "revision": saved.Revision}, true)
	status(t, w, 200)
	json.Unmarshal(w.Body.Bytes(), &saved)
	c.app.db.Close()
	c.app.db, err = openDatabase(filepath.Join(c.app.cfg.Data, "mirrorouc.db"))
	if err != nil {
		t.Fatal(err)
	}
	record, _ := c.app.record("settings", "site")
	var restored Settings
	decodeData(record.Data, &restored)
	if !reflect.DeepEqual(restored.Downloads, settings.Downloads) {
		t.Fatal("catalog changed after an older form save or process restart")
	}
	settings.Downloads = []Download{}
	w = c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": saved.Revision}, true)
	status(t, w, 200)
	json.Unmarshal(w.Body.Bytes(), &saved)
	decodeData(saved.Data, &restored)
	if restored.Downloads == nil || len(restored.Downloads) != 0 {
		t.Fatal("explicitly clearing the catalog was not preserved")
	}
}

func TestDownloadCatalogValidationAndPermissions(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	record, _ := c.app.record("settings", "site")
	base := Download{ID: "ubuntu", Name: "Ubuntu", Category: "os", Files: []DownloadFile{{Label: "24.04 LTS (amd64)", URL: "/mirrors/ubuntu-releases/24.04/ubuntu.iso"}}}
	mutations := []func(*Download){
		func(d *Download) { d.ID = "" },
		func(d *Download) { d.Category = "unknown" },
		func(d *Download) { d.Name = " " },
		func(d *Download) { d.Files = nil },
		func(d *Download) { d.Files[0].Label = " " },
		func(d *Download) { d.Files[0].URL = "javascript:alert(1)" },
		func(d *Download) { d.Files[0].URL = "//example.invalid/a.iso" },
		func(d *Download) { d.Files[0].URL = "/\\example.invalid/a.iso" },
		func(d *Download) { d.Files[0].URL = "https://user:password@example.invalid/a.iso" },
		func(d *Download) { d.URL = "https://example.invalid/ambiguous-old-field" },
	}
	for index, mutate := range mutations {
		var item Download
		decodeData(base, &item)
		mutate(&item)
		settings := initialSettings()
		settings.Downloads = []Download{item}
		w := c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": record.Revision}, true)
		if w.Code != 400 {
			t.Fatalf("invalid download case %d: status %d", index, w.Code)
		}
	}
	settings := initialSettings()
	settings.Downloads = []Download{base, base}
	status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": record.Revision}, true), 400)
	settings.Downloads = []Download{base}
	c.login("editor")
	status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": record.Revision}, true), 403)
}
