package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

//go:embed generator-defaults.json
var generatorDefaultsJSON []byte

type GeneratorRelease struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Stable    *bool  `json:"stable,omitempty"`
	Firmware  *bool  `json:"firmware,omitempty"`
	Backports *bool  `json:"backports,omitempty"`
	Keyring   string `json:"keyring,omitempty"`
}

type GeneratorVersions struct {
	DefaultRelease string             `json:"defaultRelease"`
	Releases       []GeneratorRelease `json:"releases"`
}

var generatorTypes = []string{"debian", "ubuntu", "kubernetes", "tailscale", "proxmox"}
var suiteID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var kubernetesID = regexp.MustCompile(`^v[1-9][0-9]{0,2}\.(0|[1-9][0-9]{0,2})$`)
var tailscaleID = regexp.MustCompile(`^(debian|ubuntu)/[a-z][a-z0-9-]{0,31}$`)
var keyringPath = regexp.MustCompile(`^/(usr/share/keyrings|etc/apt/trusted\.gpg\.d)/[a-zA-Z0-9][a-zA-Z0-9_-]{0,95}\.gpg$`)

func generatorType(id string) bool {
	for _, known := range generatorTypes {
		if id == known {
			return true
		}
	}
	return false
}

// Values are inserted into repository files and shell commands. Accept only
// version identifiers and keyring paths, never arbitrary templates or scripts.
func validGeneratorVersions(id string, data GeneratorVersions) bool {
	if !generatorType(id) || len(data.Releases) == 0 || len(data.Releases) > 100 {
		return false
	}
	seen := map[string]bool{}
	hasDefault, hasStable := false, false
	for _, release := range data.Releases {
		validID := suiteID.MatchString(release.ID)
		if id == "kubernetes" {
			validID = kubernetesID.MatchString(release.ID)
		}
		if id == "tailscale" {
			validID = tailscaleID.MatchString(release.ID)
		}
		if !validID || seen[release.ID] || strings.TrimSpace(release.Label) == "" || len([]rune(release.Label)) > 160 || strings.IndexFunc(release.Label, unicode.IsControl) >= 0 {
			return false
		}
		seen[release.ID] = true
		hasDefault = hasDefault || release.ID == data.DefaultRelease
		if id == "debian" {
			if release.Stable == nil || release.Firmware == nil || release.Backports == nil || (*release.Backports && !*release.Stable) {
				return false
			}
			hasStable = hasStable || *release.Stable
		} else if release.Stable != nil || release.Firmware != nil || release.Backports != nil {
			return false
		}
		if id == "proxmox" {
			if !keyringPath.MatchString(release.Keyring) {
				return false
			}
		} else if release.Keyring != "" {
			return false
		}
	}
	return hasDefault && (id != "debian" || hasStable)
}

// Seed only missing records. Subsequent releases never overwrite admin edits.
func (a *App) seedGeneratorVersions() error {
	var defaults map[string]GeneratorVersions
	if err := json.Unmarshal(generatorDefaultsJSON, &defaults); err != nil {
		return err
	}
	for _, id := range generatorTypes {
		data := defaults[id]
		if !validGeneratorVersions(id, data) {
			return errors.New("invalid generator defaults: " + id)
		}
		if _, err := a.record("generators", id); errors.Is(err, errMissing) {
			if _, err = a.saveRecord("generators", id, data, 0); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (a *App) generatorCatalog(w http.ResponseWriter, r *http.Request) {
	records, err := a.records("generators")
	if err != nil {
		a.internal(w, err)
		return
	}
	catalog := map[string]GeneratorVersions{}
	for _, record := range records {
		var data GeneratorVersions
		if decodeData(record.Data, &data) != nil || !validGeneratorVersions(record.ID, data) {
			a.internal(w, errors.New("invalid generator record: "+record.ID))
			return
		}
		catalog[record.ID] = data
	}
	respond(w, 200, catalog)
}
