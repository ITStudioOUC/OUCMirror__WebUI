package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

var errConflict = errors.New("CONFLICT")
var errMissing = errors.New("NOT_FOUND")

func openDatabase(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users(id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, password TEXT NOT NULL, role TEXT NOT NULL CHECK(role IN ('admin','editor')), created INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS sessions(token TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, csrf TEXT NOT NULL, expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS records(kind TEXT NOT NULL,id TEXT NOT NULL,data TEXT NOT NULL,revision INTEGER NOT NULL DEFAULT 1,updated INTEGER NOT NULL,deleted INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(kind,id));
CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY,username TEXT NOT NULL,action TEXT NOT NULL,target TEXT NOT NULL,result TEXT NOT NULL,created INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS config_versions(id TEXT PRIMARY KEY,files TEXT NOT NULL,base_hash TEXT NOT NULL,status TEXT NOT NULL,created INTEGER NOT NULL,username TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS cache(key TEXT PRIMARY KEY,data TEXT NOT NULL,updated INTEGER NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err = migrateSharedContent(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
func (a *App) record(kind, id string) (Record, error) {
	var r Record
	var raw string
	r.ID = id
	err := a.db.QueryRow("SELECT data,revision,updated FROM records WHERE kind=? AND id=? AND deleted=0", kind, id).Scan(&raw, &r.Revision, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return r, errMissing
	}
	if err != nil {
		return r, err
	}
	err = json.Unmarshal([]byte(raw), &r.Data)
	if kind == "settings" && id == "site" {
		r.Data = settingsDefaults(r.Data)
	}
	return r, err
}
func (a *App) records(kind string) ([]Record, error) {
	rows, err := a.db.Query("SELECT id,data,revision,updated FROM records WHERE kind=? AND deleted=0 ORDER BY updated DESC,id", kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		var r Record
		var raw string
		if err = rows.Scan(&r.ID, &raw, &r.Revision, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &r.Data); err != nil {
			return nil, err
		}
		if kind == "settings" && r.ID == "site" {
			r.Data = settingsDefaults(r.Data)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (a *App) saveRecord(kind, id string, data any, revision int) (Record, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Record{}, err
	}
	now := time.Now().UnixMilli()
	if revision == 0 {
		_, err = a.db.Exec("INSERT INTO records(kind,id,data,revision,updated) VALUES(?,?,?,1,?)", kind, id, string(raw), now)
		if err != nil {
			return Record{}, errConflict
		}
	} else {
		var result sql.Result
		result, err = a.db.Exec("UPDATE records SET data=?,revision=revision+1,updated=?,deleted=0 WHERE kind=? AND id=? AND revision=? AND deleted=0", string(raw), now, kind, id, revision)
		if err != nil {
			return Record{}, err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return Record{}, errConflict
		}
	}
	return a.record(kind, id)
}
func (a *App) audit(s Session, action, target, result string) {
	_, err := a.db.Exec("INSERT INTO audit(username,action,target,result,created) VALUES(?,?,?,?,?)", s.Username, action, target, result, time.Now().UnixMilli())
	if err != nil {
		a.logger.Printf("audit write failed: %v", err)
	}
}
func decodeData(value any, target any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
