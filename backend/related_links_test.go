package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRelatedLinksPersistIndependentlyAndSupportOlderForms(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	initial, _ := c.app.record("settings", "site")
	var legacy map[string]any
	decodeData(initial.Data, &legacy)
	delete(legacy, "relatedLinks")
	if _, err := c.app.saveRecord("settings", "site", legacy, initial.Revision); err != nil {
		t.Fatal(err)
	}
	current, _ := c.app.record("settings", "site")
	var settings Settings
	decodeData(current.Data, &settings)
	if len(settings.RelatedLinks) != 3 || settings.RelatedLinks[0].Kind != "help" {
		t.Fatal("legacy sidebar defaults missing")
	}
	response := c.call("GET", "/api/v1/admin/records/settings", nil, false)
	var adminRecords []struct{ Data Settings }
	json.Unmarshal(response.Body.Bytes(), &adminRecords)
	if len(adminRecords) != 1 || !reflect.DeepEqual(adminRecords[0].Data.RelatedLinks, settings.RelatedLinks) {
		t.Fatal("admin/public defaults differ")
	}
	footer := append([]Link{}, settings.Links...)
	settings.RelatedLinks = []RelatedLink{
		{Title: Localized{"zh-CN": "站点状态", "en-US": "Status"}, Kind: "url", URL: "/status"},
		{Title: Localized{"zh-CN": "使用帮助"}, Kind: "help", URL: ""},
	}
	response = c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": current.Revision}, true)
	status(t, response, 200)
	var saved Record
	json.Unmarshal(response.Body.Bytes(), &saved)
	legacy = nil
	decodeData(saved.Data, &legacy)
	delete(legacy, "relatedLinks")
	response = c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": legacy, "revision": saved.Revision}, true)
	status(t, response, 200)
	json.Unmarshal(response.Body.Bytes(), &saved)
	response = c.call("GET", "/api/v1/public", nil, false)
	var public struct{ Site Settings }
	json.Unmarshal(response.Body.Bytes(), &public)
	if !reflect.DeepEqual(public.Site.RelatedLinks, settings.RelatedLinks) || !reflect.DeepEqual(public.Site.Links, footer) {
		t.Fatal("sidebar edits changed footer links or were lost by older form")
	}
	settings.RelatedLinks = []RelatedLink{}
	status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": saved.Revision}, true), 200)
	c.app.db.Close()
	var err error
	c.app.db, err = openDatabase(filepath.Join(c.app.cfg.Data, "mirrorouc.db"))
	if err != nil {
		t.Fatal(err)
	}
	response = c.call("GET", "/api/v1/public", nil, false)
	public.Site = Settings{}
	json.Unmarshal(response.Body.Bytes(), &public)
	if public.Site.RelatedLinks == nil || len(public.Site.RelatedLinks) != 0 {
		t.Fatal("explicitly cleared sidebar reset after restart")
	}
}

func TestRelatedLinkValidation(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	record, _ := c.app.record("settings", "site")
	settings := initialSettings()
	bad := []RelatedLink{
		{Title: Localized{"zh-CN": "测试"}, Kind: "url", URL: "javascript:alert(1)"},
		{Title: Localized{"zh-CN": "测试"}, Kind: "url", URL: "data:text/html,example"},
		{Title: Localized{"zh-CN": "测试"}, Kind: "url", URL: "//example.invalid"},
		{Title: Localized{"zh-CN": "测试"}, Kind: "url", URL: "/\\example.invalid"},
		{Title: Localized{"zh-CN": "测试"}, Kind: "shell", URL: "/status"},
		{Title: Localized{"zh-CN": "测试"}, Kind: "help", URL: "https://example.invalid/"},
		{Title: Localized{"zh-CN": " "}, Kind: "url", URL: "https://example.invalid/"},
	}
	for _, link := range bad {
		settings.RelatedLinks = []RelatedLink{link}
		status(t, c.call("PUT", "/api/v1/admin/records/settings/site", map[string]any{"data": settings, "revision": record.Revision}, true), 400)
	}
	for _, value := range []string{"https://example.invalid/", "http://example.invalid/", "/status", "/status?range=1#monitor-sync"} {
		if !navigationURL(value) {
			t.Fatalf("valid navigation URL rejected: %s", value)
		}
	}
	c.login("editor")
	status(t, c.call("GET", "/api/v1/admin/records/settings", nil, false), 403)
}
