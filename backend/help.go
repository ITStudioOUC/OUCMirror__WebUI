package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type helpArticle struct {
	Name        string    `json:"name"`
	Description Localized `json:"description"`
	Website     string    `json:"website"`
	BrowseURL   string    `json:"browseUrl"`
	Body        string    `json:"body,omitempty"`
}

func publishedHelp(meta MirrorMetadata) bool {
	return meta.Visible && strings.TrimSpace(meta.Help) != ""
}

func helpContent(meta MirrorMetadata, body bool) helpArticle {
	article := helpArticle{Name: meta.Name, Description: meta.Description, Website: meta.Website, BrowseURL: "/mirrors/" + meta.Name + "/"}
	if body {
		article.Body = meta.Help
	}
	return article
}

func (a *App) findHelp(name string) (MirrorMetadata, error) {
	var meta MirrorMetadata
	if !validName(name) {
		return meta, errMissing
	}
	record, err := a.record("mirrors", name)
	if err != nil {
		return meta, err
	}
	if err := decodeData(record.Data, &meta); err != nil {
		return meta, err
	}
	if meta.Name != name || !publishedHelp(meta) {
		return meta, errMissing
	}
	return meta, nil
}

func (a *App) helpIndex(w http.ResponseWriter, r *http.Request) {
	records, err := a.records("mirrors")
	if err != nil {
		a.internal(w, err)
		return
	}
	articles := []helpArticle{}
	for _, record := range records {
		var meta MirrorMetadata
		if decodeData(record.Data, &meta) == nil && validName(meta.Name) && meta.Name == record.ID && publishedHelp(meta) {
			articles = append(articles, helpContent(meta, false))
		}
	}
	sort.Slice(articles, func(i, j int) bool { return articles[i].Name < articles[j].Name })
	respond(w, 200, articles)
}

func (a *App) helpArticle(w http.ResponseWriter, r *http.Request) {
	meta, err := a.findHelp(r.PathValue("name"))
	if errors.Is(err, errMissing) {
		fail(w, 404, "NOT_FOUND")
		return
	}
	if err != nil {
		a.internal(w, err)
		return
	}
	respond(w, 200, helpContent(meta, true))
}

func (a *App) helpPage(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	index := filepath.Join(a.cfg.Static, "index.html")
	w.Header().Set("Cache-Control", "no-cache")
	if path == "/help" {
		http.ServeFile(w, r, index)
		return
	}
	_, err := a.findHelp(strings.TrimPrefix(path, "/help/"))
	if err == nil {
		http.ServeFile(w, r, index)
		return
	}
	if !errors.Is(err, errMissing) {
		a.internal(w, err)
		return
	}
	// Return the application for a friendly error page while retaining a real 404.
	raw, err := os.ReadFile(index)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if r.Method != http.MethodHead {
		w.Write(raw)
	}
}
