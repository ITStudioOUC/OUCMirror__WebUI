package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Normalize on read, leaving original database rows intact until an administrator saves.
func normalizeDownloads(input []Download) []Download {
	output := []Download{}
	legacy := map[string]int{}
	for _, item := range input {
		if item.ID == "" && item.Category == "" && item.Description == "" && item.Files == nil && (item.URL != "" || item.Version != "") {
			name := strings.TrimSpace(item.Name)
			index, exists := legacy[name]
			if !exists {
				hash := sha256.Sum256([]byte(name))
				index = len(output)
				legacy[name] = index
				output = append(output, Download{ID: fmt.Sprintf("legacy-%x", hash[:12]), Name: name, Category: "software", Files: []DownloadFile{}})
			}
			label := strings.TrimSpace(item.Version)
			if label == "" {
				label = name
			}
			output[index].Files = append(output[index].Files, DownloadFile{Label: label, URL: item.URL})
		} else {
			output = append(output, item)
		}
	}
	return output
}

func validDownloads(items []Download) bool {
	if len(items) > 100 {
		return false
	}
	ids := map[string]bool{}
	for _, item := range items {
		if !validName(item.ID) || ids[item.ID] || strings.TrimSpace(item.Name) == "" || len(item.Name) > 300 || len(item.Description) > 100000 || len(item.Files) < 1 || len(item.Files) > 100 || item.Version != "" || item.URL != "" {
			return false
		}
		if item.Category != "os" && item.Category != "software" && item.Category != "fonts" {
			return false
		}
		ids[item.ID] = true
		for _, file := range item.Files {
			if strings.TrimSpace(file.Label) == "" || len(file.Label) > 500 || !navigationURL(file.URL) {
				return false
			}
		}
	}
	return true
}
