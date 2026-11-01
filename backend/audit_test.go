package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestAuditPagination(t *testing.T) {
	c := fixture(t)
	c.login("admin")
	type entry struct {
		ID int `json:"id"`
	}
	type auditPage struct {
		Items      []entry `json:"items"`
		Page       int     `json:"page"`
		PageSize   int     `json:"pageSize"`
		Total      int     `json:"total"`
		TotalPages int     `json:"totalPages"`
	}
	readPage := func(t *testing.T, page int) auditPage {
		t.Helper()
		w := c.call("GET", fmt.Sprintf("/api/v1/admin/audit?page=%d", page), nil, false)
		status(t, w, 200)
		var result auditPage
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	for _, count := range []int{0, 10, 11, 205} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			tx, err := c.app.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if _, err := tx.Exec("DELETE FROM audit"); err != nil {
				t.Fatal(err)
			}
			for id := 1; id <= count; id++ {
				if _, err := tx.Exec("INSERT INTO audit(id,username,action,target,result,created) VALUES(?,'admin','record_save','settings/site','success',?)", id, id); err != nil {
					t.Fatal(err)
				}
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			pages, seen := max(1, (count+9)/10), 0
			for page := 1; page <= pages; page++ {
				result := readPage(t, page)
				wantRows := min(10, count-seen)
				if result.Items == nil || result.Page != page || result.PageSize != 10 || result.Total != count || result.TotalPages != pages || len(result.Items) != wantRows {
					t.Fatalf("incorrect pagination for page %d: %+v", page, result)
				}
				for _, item := range result.Items {
					if item.ID != count-seen {
						t.Fatalf("duplicate, skipped or unordered record: want %d, got %d", count-seen, item.ID)
					}
					seen++
				}
			}
			if seen != count {
				t.Fatal("older records are inaccessible")
			}
			last := readPage(t, 999999)
			if last.Page != pages || len(last.Items) != max(0, count-(pages-1)*10) {
				t.Fatalf("out-of-range page was not clamped: %+v", last)
			}
			w := c.call("GET", "/api/v1/admin/audit", nil, false)
			status(t, w, 200)
			var legacy []entry
			if err := json.Unmarshal(w.Body.Bytes(), &legacy); err != nil || len(legacy) != min(200, count) {
				t.Fatal("already-open older admin pages lost their array response")
			}
		})
	}
	for _, page := range []string{"", "0", "-1", "abc", "1.5", "99999999999999999999999"} {
		status(t, c.call("GET", "/api/v1/admin/audit?page="+page, nil, false), 400)
	}
}

func TestAuditPaginationRequiresAdmin(t *testing.T) {
	c := fixture(t)
	status(t, c.call("GET", "/api/v1/admin/audit?page=1", nil, false), 401)
	c.login("editor")
	status(t, c.call("GET", "/api/v1/admin/audit?page=1", nil, false), 403)
}
