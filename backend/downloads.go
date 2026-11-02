package main

import (
	"crypto/sha256"
	"fmt"
	"html"
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
	for index, item := range output {
		if len(item.Files) == 0 || len(item.Files) > 100 {
			continue
		}
		valid := true
		for _, file := range item.Files {
			if strings.TrimSpace(file.Label) == "" || len(file.Label) > 500 || !navigationURL(file.URL) {
				valid = false
				break
			}
		}
		if !valid {
			continue // Keep invalid legacy fields so writes are rejected rather than silently dropping them.
		}
		links := make([]string, 0, len(item.Files))
		for _, file := range item.Files {
			label := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]", "\r", " ", "\n", " ").Replace(html.EscapeString(file.Label))
			url := strings.NewReplacer("<", "%3C", ">", "%3E", "\\", "%5C", " ", "%20").Replace(file.URL)
			if strings.Contains(item.Description, "]("+file.URL+")") || strings.Contains(item.Description, "](<"+file.URL+">)") || strings.Contains(item.Description, "]("+url+")") || strings.Contains(item.Description, "](<"+url+">)") {
				continue
			}
			links = append(links, fmt.Sprintf("- [%s](<%s>)", label, url))
		}
		if len(links) > 0 {
			if item.Description != "" {
				item.Description += "\n\n"
			}
			item.Description += strings.Join(links, "\n")
		}
		item.Files = nil
		output[index] = item
	}
	return output
}

func validDownloads(items []Download) bool {
	if len(items) > 100 {
		return false
	}
	ids := map[string]bool{}
	for _, item := range items {
		if !validName(item.ID) || ids[item.ID] || strings.TrimSpace(item.Name) == "" || len(item.Name) > 300 || strings.TrimSpace(item.Description) == "" || len(item.Description) > 100000 || len(item.Files) != 0 || item.Version != "" || item.URL != "" {
			return false
		}
		if item.Category != "os" && item.Category != "software" && item.Category != "fonts" {
			return false
		}
		ids[item.ID] = true
	}
	return true
}
