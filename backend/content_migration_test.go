package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSharedChineseContentMigration(t *testing.T) {
	c := fixture(t)
	legacy := map[string]any{"title": Localized{"zh-CN": "中文公告", "en-US": "English title"}, "body": Localized{"zh-CN": "# 中文正文", "en-US": "English body"}, "published": true, "pinned": false, "date": time.Now().UTC().Format(time.RFC3339)}
	if _, err := c.app.saveRecord("announcements", "legacy", legacy, 0); err != nil {
		t.Fatal(err)
	}
	metadata := map[string]any{"name": "alpine", "description": Localized{"zh-CN": "介绍", "en-US": "Description"}, "help": Localized{"zh-CN": "# 中文帮助", "en-US": "English help"}, "category": "os", "visible": true, "website": ""}
	if _, err := c.app.saveRecord("mirrors", "alpine", metadata, 0); err != nil {
		t.Fatal(err)
	}
	if err := migrateSharedContent(c.app.db); err != nil {
		t.Fatal(err)
	}
	record, _ := c.app.record("announcements", "legacy")
	var article Announcement
	if err := decodeData(record.Data, &article); err != nil {
		t.Fatal(err)
	}
	if article.Title != "中文公告" || article.Body != "# 中文正文" || record.Revision != 2 {
		t.Fatal("Chinese edition not preserved")
	}
	record, _ = c.app.record("mirrors", "alpine")
	var mirror MirrorMetadata
	if err := decodeData(record.Data, &mirror); err != nil {
		t.Fatal(err)
	}
	if mirror.Help != "# 中文帮助" || mirror.Description["en-US"] != "Description" {
		t.Fatal("help migration changed unrelated metadata")
	}
	var backup string
	if err := c.app.db.QueryRow("SELECT data FROM shared_content_backups WHERE kind='announcements' AND id='legacy'").Scan(&backup); err != nil || !strings.Contains(backup, "English body") {
		t.Fatal("original language editions were not backed up")
	}
	if err := migrateSharedContent(c.app.db); err != nil {
		t.Fatal(err)
	}
	record, _ = c.app.record("announcements", "legacy")
	if record.Revision != 2 {
		t.Fatal("migration is not idempotent")
	}
	response := c.call("GET", "/api/v1/public", nil, false)
	var payload struct{ Announcements []struct{ Data Announcement } }
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || len(payload.Announcements) != 1 || payload.Announcements[0].Data.Title != "中文公告" {
		t.Fatal("public API did not expose shared strings")
	}
	c.login("admin")
	status(t, c.call("PUT", "/api/v1/admin/records/announcements/legacy", map[string]any{"revision": 2, "data": legacy}, true), 400)
	article.Body = "   "
	status(t, c.call("PUT", "/api/v1/admin/records/announcements/legacy", map[string]any{"revision": 2, "data": article}, true), 400)
}

func TestSharedContentMigrationKeepsEmptyChineseAndLegacyOnlyContent(t *testing.T) {
	c := fixture(t)
	for name, help := range map[string]any{"empty-zh": Localized{"zh-CN": "", "en-US": "Alternate text"}, "legacy-only": Localized{"en-US": "Existing content"}, "missing": nil} {
		if _, err := c.app.saveRecord("mirrors", name, map[string]any{"name": name, "help": help}, 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateSharedContent(c.app.db); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"empty-zh": "", "legacy-only": "Existing content", "missing": ""} {
		record, _ := c.app.record("mirrors", name)
		var mirror MirrorMetadata
		if err := decodeData(record.Data, &mirror); err != nil || mirror.Help != want {
			t.Fatalf("unexpected migrated help for %s", name)
		}
	}
}
