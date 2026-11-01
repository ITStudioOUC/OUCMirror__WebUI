package main

import (
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// Content is a single shared Chinese edition; interface labels remain localized.
// Keep every original row before converting legacy language maps, including deleted rows.
func migrateSharedContent(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`CREATE TABLE IF NOT EXISTS shared_content_backups(kind TEXT NOT NULL,id TEXT NOT NULL,revision INTEGER NOT NULL,data TEXT NOT NULL,created INTEGER NOT NULL,PRIMARY KEY(kind,id,revision))`); err != nil {
		return err
	}
	rows, err := tx.Query("SELECT kind,id,data,revision FROM records WHERE kind IN ('announcements','mirrors')")
	if err != nil {
		return err
	}
	type legacyRow struct {
		kind, id, before, after string
		revision                int
	}
	changes := []legacyRow{}
	for rows.Next() {
		var row legacyRow
		if err = rows.Scan(&row.kind, &row.id, &row.before, &row.revision); err != nil {
			rows.Close()
			return err
		}
		var data map[string]json.RawMessage
		if err = json.Unmarshal([]byte(row.before), &data); err != nil {
			rows.Close()
			return err
		}
		fields := []string{"help"}
		if row.kind == "announcements" {
			fields = []string{"title", "body"}
		}
		changed := false
		for _, field := range fields {
			value, exists := data[field]
			if !exists {
				data[field] = json.RawMessage(`""`)
				changed = true
				continue
			}
			var single string
			if string(value) != "null" && json.Unmarshal(value, &single) == nil {
				continue
			}
			var localized map[string]string
			if err = json.Unmarshal(value, &localized); err != nil {
				rows.Close()
				return err
			}
			// A present Chinese edition wins, including an intentionally empty one.
			single, exists = localized["zh-CN"]
			if !exists {
				single, exists = localized["zh"]
			}
			if !exists {
				keys := make([]string, 0, len(localized))
				for key := range localized {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					if strings.TrimSpace(localized[key]) != "" {
						single = localized[key]
						break
					}
				}
			}
			data[field], _ = json.Marshal(single)
			changed = true
		}
		if changed {
			raw, e := json.Marshal(data)
			if e != nil {
				rows.Close()
				return e
			}
			row.after = string(raw)
			changes = append(changes, row)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, row := range changes {
		now := time.Now().UnixMilli()
		if _, err = tx.Exec("INSERT INTO shared_content_backups(kind,id,revision,data,created) VALUES(?,?,?,?,?)", row.kind, row.id, row.revision, row.before, now); err != nil {
			return err
		}
		if _, err = tx.Exec("UPDATE records SET data=?,revision=revision+1,updated=? WHERE kind=? AND id=? AND revision=?", row.after, now, row.kind, row.id, row.revision); err != nil {
			return err
		}
	}
	return tx.Commit()
}
