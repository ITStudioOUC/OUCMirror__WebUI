package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGeneratorCatalogPersistenceAndAuthorization(t *testing.T) {
	c := fixture(t)
	w := c.call("GET", "/api/v1/generators", nil, false)
	status(t, w, 200)
	var catalog map[string]GeneratorVersions
	json.Unmarshal(w.Body.Bytes(), &catalog)
	if len(catalog) != 5 {
		t.Fatal("missing seeded version catalogs")
	}
	settingsBefore, _ := c.app.record("settings", "site")
	before, _ := c.app.record("generators", "ubuntu")
	versions := catalog["ubuntu"]
	// Deliberately fictional future data: no official codename is assumed.
	versions.Releases = append(versions.Releases, GeneratorRelease{ID: "future-test", Label: "Ubuntu future (test fixture)"})
	versions.DefaultRelease = "future-test"
	body := map[string]any{"data": versions, "revision": before.Revision}
	path := "/api/v1/admin/records/generators/ubuntu"
	status(t, c.call("PUT", path, body, true), 401)
	c.login("editor")
	status(t, c.call("GET", "/api/v1/admin/records/generators", nil, false), 403)
	status(t, c.call("PUT", path, body, true), 403)
	c.login("admin")
	status(t, c.call("PUT", path, body, false), 403)
	status(t, c.call("PUT", path, body, true), 200)
	status(t, c.call("PUT", path, body, true), 409)
	status(t, c.call("DELETE", path, map[string]any{"revision": before.Revision + 1}, true), 400)
	// Starting a new app on the same DB must not overwrite edited defaults.
	c.app.db.Close()
	restarted, err := newApp(c.app.cfg)
	if err != nil {
		t.Fatal(err)
	}
	c.app = restarted
	t.Cleanup(func() { restarted.db.Close() })
	w = c.call("GET", "/api/v1/generators", nil, false)
	status(t, w, 200)
	json.Unmarshal(w.Body.Bytes(), &catalog)
	if !reflect.DeepEqual(catalog["ubuntu"], versions) {
		t.Fatal("saved future release lost after restart")
	}
	settingsAfter, _ := c.app.record("settings", "site")
	if !reflect.DeepEqual(settingsBefore, settingsAfter) {
		t.Fatal("generator edits changed site settings")
	}
	var records []Record
	w = c.call("GET", "/api/v1/admin/records/generators", nil, false)
	status(t, w, 200)
	json.Unmarshal(w.Body.Bytes(), &records)
	if len(records) != 5 {
		t.Fatal("admin cannot read saved version catalogs")
	}
}

func TestGeneratorCatalogRejectsUnsafeAndInconsistentData(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	var defaults map[string]GeneratorVersions
	json.Unmarshal(generatorDefaultsJSON, &defaults)
	for _, id := range generatorTypes {
		if !validGeneratorVersions(id, defaults[id]) {
			t.Fatal("invalid shared seed", id)
		}
	}
	for _, id := range []string{"ubuntu", "kubernetes", "tailscale", "proxmox", "debian"} {
		for _, bad := range []string{"$(id)", "noble\ntrusted=yes", "../../etc/passwd", "foo;id", "foo bar", "__proto__"} {
			data := GeneratorVersions{DefaultRelease: bad, Releases: []GeneratorRelease{{ID: bad, Label: "test"}}}
			if validGeneratorVersions(id, data) {
				t.Fatal("accepted unsafe ID", id, bad)
			}
		}
	}
	invalid := []any{
		map[string]any{"defaultRelease": "noble", "releases": []any{}},
		map[string]any{"defaultRelease": "missing", "releases": []GeneratorRelease{{ID: "noble", Label: "Ubuntu"}}},
		map[string]any{"defaultRelease": "noble", "releases": []GeneratorRelease{{ID: "noble", Label: "Ubuntu"}, {ID: "noble", Label: "duplicate"}}},
		map[string]any{"defaultRelease": "noble", "releases": []GeneratorRelease{{ID: "noble", Label: ""}}},
		map[string]any{"defaultRelease": "noble", "releases": []any{map[string]any{"id": "noble", "label": "Ubuntu", "command": "id"}}},
		map[string]any{"defaultRelease": "noble", "releases": []GeneratorRelease{{ID: "noble", Label: "Ubuntu", Keyring: "/tmp/other.gpg"}}},
	}
	record, _ := c.app.record("generators", "ubuntu")
	for _, data := range invalid {
		status(t, c.call("PUT", "/api/v1/admin/records/generators/ubuntu", map[string]any{"data": data, "revision": record.Revision}, true), 400)
	}
	status(t, c.call("PUT", "/api/v1/admin/records/generators/unknown", map[string]any{"data": defaults["ubuntu"], "revision": 0}, true), 400)
	pve := defaults["proxmox"]
	for _, path := range []string{"/tmp/a.gpg", "/usr/share/keyrings/../../tmp/a.gpg", "/usr/share/keyrings/a.gpg\nmalicious", "/usr/share/keyrings/$(id).gpg"} {
		pve.Releases[0].Keyring = path
		if validGeneratorVersions("proxmox", pve) {
			t.Fatal("accepted unsafe keyring", path)
		}
	}
	debian := defaults["debian"]
	for index := range debian.Releases {
		value := false
		debian.Releases[index].Stable = &value
		debian.Releases[index].Backports = &value
	}
	if validGeneratorVersions("debian", debian) {
		t.Fatal("Debian Security lost all stable releases")
	}
}
