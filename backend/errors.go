package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// The generated page needs no API calls and preserves the actual failure status.
func (a *App) errorPage(w http.ResponseWriter, r *http.Request, status int) {
	raw, err := os.ReadFile(filepath.Join(a.cfg.Static, "errors", strconv.Itoa(status)+".html"))
	if err != nil {
		http.Error(w, http.StatusText(status), status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write(raw)
	}
}

func (a *App) servePageFile(w http.ResponseWriter, r *http.Request, path string) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		a.errorPage(w, r, http.StatusNotFound)
		return
	}
	if errors.Is(err, os.ErrPermission) || (err == nil && info.IsDir()) {
		a.errorPage(w, r, http.StatusForbidden)
		return
	}
	if err != nil {
		a.logger.Printf("file unavailable: %v", err)
		a.errorPage(w, r, http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, path)
}
