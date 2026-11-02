package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStyledBrowserErrorsRetainStatusAndAPIJSON(t *testing.T) {
	c := fixture(t)
	dir := filepath.Join(c.app.cfg.Static, "errors")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, code := range []int{403, 404, 405, 500, 502, 503, 504} {
		html := fmt.Sprintf("<!doctype html><title>MirrorITS %d</title><h1>%d</h1>", code, code)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%d.html", code)), []byte(html), 0600); err != nil {
			t.Fatal(err)
		}
		for _, method := range []string{"GET", "HEAD"} {
			w := httptest.NewRecorder()
			c.app.errorPage(w, httptest.NewRequest(method, "/missing?code=200", nil), code)
			if w.Code != code || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("error response lost status or headers: %+v", w)
			}
			if method == "HEAD" && w.Body.Len() != 0 {
				t.Fatal("HEAD error includes a body")
			}
			if method == "GET" && w.Body.String() != html {
				t.Fatal("error response did not use the static page")
			}
		}
	}
	for _, path := range []string{"/missing-page", "/assets/missing.js", "/uploads/missing.png", "/uploads/mirrorouc.db"} {
		w := c.call("GET", path, nil, false)
		status(t, w, 404)
		if !strings.Contains(w.Body.String(), "MirrorITS 404") {
			t.Fatalf("default browser error remains at %s", path)
		}
	}
	w := c.call("POST", "/missing-page", nil, false)
	status(t, w, 405)
	if w.Header().Get("Allow") != "GET, HEAD" {
		t.Fatal("Allow header was lost")
	}
	for _, path := range []string{"/api/missing", "/api/v1/help/missing"} {
		w = c.call("GET", path, nil, false)
		status(t, w, 404)
		if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") || !strings.Contains(w.Body.String(), `"NOT_FOUND"`) {
			t.Fatal("API error changed to an HTML page")
		}
	}
	// Healthy assets still support byte ranges through the ordinary file server.
	assets := filepath.Join(c.app.cfg.Static, "assets")
	os.MkdirAll(assets, 0700)
	os.WriteFile(filepath.Join(assets, "sample.txt"), []byte("0123456789"), 0600)
	r := httptest.NewRequest("GET", "/assets/sample.txt", nil)
	r.Header.Set("Range", "bytes=2-5")
	response := httptest.NewRecorder()
	c.app.handler().ServeHTTP(response, r)
	if response.Code != http.StatusPartialContent || response.Body.String() != "2345" {
		t.Fatal("error pages changed healthy file range handling")
	}
}
