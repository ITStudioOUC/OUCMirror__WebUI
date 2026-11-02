package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
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
	if len(items) != 2 || items[0].Name != "Ubuntu" || len(items[0].Files) != 0 || !strings.Contains(items[0].Description, "[24.04 (arm64)](<https://example.invalid/ubuntu-arm.iso>)") || !strings.Contains(items[0].Description, "[24.04 (amd64)](<https://example.invalid/ubuntu.iso>)") || !strings.Contains(items[1].Description, "[Tool](<https://example.invalid/tool.zip>)") || items[0].Category != "software" || !validName(items[0].ID) || items[0].URL != "" {
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
	settings.Downloads[0].Description = "## Ubuntu\n\n[24.04 (amd64)](/mirrors/ubuntu-releases/24.04/ubuntu.iso)\n\nSHA256: `0123456789abcdef`"
	settings.Downloads = append(settings.Downloads, Download{ID: "fonts", Name: "字体", Category: "fonts", Description: "字体的安装方法。"})
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
	base := Download{ID: "ubuntu", Name: "Ubuntu", Category: "os", Description: "## Ubuntu\n\n[24.04 LTS (amd64)](/mirrors/ubuntu-releases/24.04/ubuntu.iso)"}
	mutations := []func(*Download){
		func(d *Download) { d.ID = "" },
		func(d *Download) { d.Category = "unknown" },
		func(d *Download) { d.Name = " " },
		func(d *Download) { d.Description = " \n\t" },
		func(d *Download) { d.Description = strings.Repeat("a", 100001) },
		func(d *Download) { d.Files = []DownloadFile{{Label: " ", URL: "/mirrors/a.iso"}} },
		func(d *Download) { d.URL = "https://example.invalid/ambiguous-old-field" },
	}
	for _, url := range []string{"javascript:alert(1)", "//example.invalid/a.iso", "/\\example.invalid/a.iso", "https://user:password@example.invalid/a.iso"} {
		mutations = append(mutations, func(d *Download) { d.Files = []DownloadFile{{Label: "ISO", URL: url}} })
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
	status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": record.Revision}, true), 200)
	c.login("editor")
	status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": record.Revision}, true), 403)
}

func TestDownloadArticlesPreserveExistingMarkdownAndLinksOnce(t *testing.T) {
	body := "## Debian 13\n\n版本、签名与 SHA256：`abc123`。\n\n[签名](/mirrors/debian-cd/SHA256SUMS.sign)"
	input := []Download{{ID: "debian", Name: "Debian", Category: "os", Description: body, Files: []DownloadFile{
		{Label: "ISO [amd64] <mirror>", URL: "/mirrors/debian-cd/image (amd64).iso?source=<mirror>"},
		{Label: "SHA256SUMS", URL: "https://example.invalid/SHA256SUMS"},
	}}}
	articles := normalizeDownloads(input)
	if len(articles[0].Files) != 0 || !validDownloads(articles) || !strings.HasPrefix(articles[0].Description, body+"\n\n") {
		t.Fatal("existing article or legacy links were lost")
	}
	for _, link := range []string{
		"- [ISO \\[amd64\\] &lt;mirror&gt;](</mirrors/debian-cd/image%20(amd64).iso?source=%3Cmirror%3E>)",
		"- [SHA256SUMS](<https://example.invalid/SHA256SUMS>)",
	} {
		if !strings.Contains(articles[0].Description, link) {
			t.Fatalf("legacy link not safely preserved: %s", articles[0].Description)
		}
	}
	if !reflect.DeepEqual(normalizeDownloads(articles), articles) || len(input[0].Files) != 2 || input[0].Description != body {
		t.Fatal("normalization duplicated links or mutated the input")
	}
	raw, _ := json.Marshal(articles)
	if strings.Contains(string(raw), `"files"`) {
		t.Fatal("article response still exposes a separate download-link module")
	}
}

func TestDownloadArticlesReuseLinksAlreadyInMarkdown(t *testing.T) {
	body := "## 下载\n\n[ISO](/mirrors/debian/image.iso)\n\n[签名](<https://example.invalid/image.iso.sig>)"
	input := []Download{{ID: "debian", Name: "Debian", Category: "os", Description: body, Files: []DownloadFile{
		{Label: "安装镜像", URL: "/mirrors/debian/image.iso"},
		{Label: "签名", URL: "https://example.invalid/image.iso.sig"},
	}}}
	output := normalizeDownloads(input)
	if !validDownloads(output) || output[0].Description != body || len(output[0].Files) != 0 {
		t.Fatal("migration duplicated links already maintained in the article")
	}
}
