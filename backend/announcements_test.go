package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func publicArticles(t *testing.T, c *testClient, home bool) []Record {
	t.Helper()
	path := "/api/v1/announcements"
	if home {
		path = "/api/v1/public"
	}
	w := c.call("GET", path, nil, false)
	status(t, w, 200)
	var records []Record
	if home {
		var payload struct {
			Announcements []Record `json:"announcements"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		records = payload.Announcements
	} else if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	return records
}

func TestAnnouncementsArchiveAndHome(t *testing.T) {
	c := fixture(t)
	// An archive must include older articles beyond the former 100-record cap.
	for i := 0; i < 105; i++ {
		_, err := c.app.saveRecord("announcements", fmt.Sprintf("history-%03d", i), Announcement{Title: "历史公告", Body: "## Markdown 正文\n\n保留全文。", Published: true, Date: "2020-01-01T00:00:00Z"}, 0)
		if err != nil {
			t.Fatal(err)
		}
	}
	for id, article := range map[string]Announcement{
		"pinned": {Title: "置顶公告", Body: "置顶正文", Published: true, Pinned: true, Date: "2019-01-01T00:00:00Z"},
		"newer":  {Title: "较晚公告", Body: "较晚正文", Published: true, Date: "2026-10-31T18:00:00Z"},
		"older":  {Title: "较早公告", Body: "较早正文", Published: true, Date: "2026-11-01T01:00:00+08:00"},
		"draft":  {Title: "未公开标题", Body: "未公开正文", Pinned: true, Date: "2027-01-01T00:00:00Z"},
	} {
		if _, err := c.app.saveRecord("announcements", id, article, 0); err != nil {
			t.Fatal(err)
		}
	}
	archive := publicArticles(t, c, false)
	if len(archive) != 108 {
		t.Fatalf("archive omitted published articles: %d", len(archive))
	}
	for i, id := range []string{"pinned", "newer", "older"} {
		if archive[i].ID != id {
			t.Fatalf("incorrect pinned / chronological order: %+v", archive[:3])
		}
	}
	for _, record := range archive {
		if record.ID == "draft" {
			t.Fatal("draft exposed in archive")
		}
	}
	var last Announcement
	if err := decodeData(archive[len(archive)-1].Data, &last); err != nil || last.Body != "## Markdown 正文\n\n保留全文。" {
		t.Fatal("archive did not retain complete Markdown")
	}
	home := publicArticles(t, c, true)
	if len(home) != 5 || home[0].ID != "pinned" || home[1].ID != "newer" {
		t.Fatalf("home should show five recent, pinned-first links: %+v", home)
	}
}

func TestAnnouncementPublishingLifecycle(t *testing.T) {
	c := fixture(t)
	c.login("editor")
	article := Announcement{Title: "维护通知", Date: "2026-11-01T00:00:00+08:00"}
	save := func(revision, want int) {
		t.Helper()
		status(t, c.call("PUT", "/api/v1/admin/records/announcements/maintenance", map[string]any{"data": article, "revision": revision}, true), want)
	}
	assertVisible := func(want int) {
		t.Helper()
		for _, home := range []bool{false, true} {
			if got := len(publicArticles(t, c, home)); got != want {
				t.Fatalf("public visibility: want %d, got %d (home=%v)", want, got, home)
			}
		}
	}
	save(0, 200) // Empty bodies may be saved as drafts.
	assertVisible(0)
	article.Published = true
	save(1, 400) // Publishing an empty body must not alter the draft.
	assertVisible(0)
	article.Body = "## 维护安排\n\n- 时间：今晚\n- 影响：短暂中断\n\n`apt update`"
	save(1, 200)
	assertVisible(1)
	save(1, 409) // Another editor cannot overwrite a newer revision.
	article.Published = false
	save(2, 200)
	assertVisible(0)
	article.Published = true
	save(3, 200)
	assertVisible(1)
	status(t, c.call("DELETE", "/api/v1/admin/records/announcements/maintenance", map[string]int{"revision": 4}, true), 200)
	assertVisible(0)
}

func TestAnnouncementsDirectNavigation(t *testing.T) {
	c := fixture(t)
	if err := os.WriteFile(filepath.Join(c.app.cfg.Static, "index.html"), []byte("<!doctype html><title>MirrorITS</title>"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/announcements", "/announcements/"} {
		for _, method := range []string{"GET", "HEAD"} {
			w := c.call(method, path, nil, false)
			status(t, w, 200)
			if method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD returned a response body")
			}
		}
	}
	status(t, c.call("POST", "/announcements", nil, false), 405)
	status(t, c.call("GET", "/announcements/missing", nil, false), 404)
}
