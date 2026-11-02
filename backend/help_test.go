package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedHelpAndDeepLinks(t *testing.T) {
	c := fixture(t)
	article := MirrorMetadata{Name: "alpine", Description: Localized{"zh-CN": "软件仓库"}, Help: "## 配置软件源\n\n```sh\napk update\n```", Category: "os", Visible: true, Website: "https://alpinelinux.org/"}
	for _, meta := range []MirrorMetadata{
		article,
		{Name: "hidden", Help: "private draft", Category: "os", Visible: false},
		{Name: "empty", Help: " \n\t", Category: "os", Visible: true},
	} {
		if _, err := c.app.saveRecord("mirrors", meta.Name, meta, 0); err != nil {
			t.Fatal(err)
		}
	}
	// Reading saved help does not depend on a successful live tunasync poll.
	c.app.jobsMu.Lock()
	c.app.jobs = nil
	c.app.jobsError = "MANAGER_UNAVAILABLE"
	c.app.jobsMu.Unlock()
	response := c.call("GET", "/api/v1/help", nil, false)
	status(t, response, 200)
	var index []helpArticle
	if err := json.Unmarshal(response.Body.Bytes(), &index); err != nil || len(index) != 1 || index[0].Name != "alpine" || index[0].Body != "" {
		t.Fatal("help index leaked hidden content or included unpublished articles")
	}
	response = c.call("GET", "/api/v1/help/alpine", nil, false)
	status(t, response, 200)
	var result helpArticle
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Body != article.Help || result.BrowseURL != "/mirrors/alpine/" {
		t.Fatal("help endpoint did not preserve the shared Chinese Markdown")
	}
	for _, name := range []string{"hidden", "empty", "missing", "bad%2Fname"} {
		status(t, c.call("GET", "/api/v1/help/"+name, nil, false), 404)
	}
	const html = "<!doctype html><title>MirrorITS help</title>"
	if err := os.WriteFile(filepath.Join(c.app.cfg.Static, "index.html"), []byte(html), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/help", "/help/", "/help/alpine", "/help/alpine/"} {
		response = c.call("GET", path, nil, false)
		status(t, response, 200)
		if response.Body.String() != html {
			t.Fatal("refreshing a help deep link did not serve the application")
		}
		status(t, c.call("HEAD", path, nil, false), 200)
	}
	for _, path := range []string{"/help/hidden/", "/help/empty/", "/help/missing/", "/help/alpine/extra"} {
		response = c.call("GET", path, nil, false)
		status(t, response, 404)
		if response.Body.String() != html || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") {
			t.Fatal("missing help should retain a real 404 and display the application error page")
		}
		response = c.call("HEAD", path, nil, false)
		status(t, response, 404)
		if response.Body.Len() != 0 {
			t.Fatal("HEAD returned a response body")
		}
	}
	status(t, c.call("POST", "/help/alpine/", nil, false), 405)
	status(t, c.call("GET", "/helpful", nil, false), 404)
}

func TestHelpPublicationUsesMirrorEditorAndPermissions(t *testing.T) {
	c := fixture(t)
	meta := MirrorMetadata{Name: "alpine", Help: "## 安装\n\n中文说明。", Category: "os", Visible: true}
	c.login("admin")
	status(t, c.call("PUT", "/api/v1/admin/records/mirrors/alpine", map[string]any{"revision": 0, "data": meta}, true), 200)
	status(t, c.call("GET", "/api/v1/help/alpine", nil, false), 200)
	c.login("editor")
	status(t, c.call("PUT", "/api/v1/admin/records/mirrors/alpine", map[string]any{"revision": 1, "data": meta}, true), 403)
	c.login("admin")
	meta.Help = ""
	status(t, c.call("PUT", "/api/v1/admin/records/mirrors/alpine", map[string]any{"revision": 1, "data": meta}, true), 200)
	status(t, c.call("GET", "/api/v1/help/alpine", nil, false), 404)
	response := c.call("GET", "/api/v1/help", nil, false)
	status(t, response, 200)
	if strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatal("clearing help did not remove its public entry")
	}
}
