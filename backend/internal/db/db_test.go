package db

import (
	"database/sql"
	"os"
	"testing"
)

// NewTestDB creates an in-memory SQLite database with the full schema applied.
// It is automatically closed when the test ends.
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open("file:testhelper?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatalf("NewTestDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.db"

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Verify schema tables exist
	tables := []string{"users", "skills", "events", "event_skills", "registrations", "attendances", "user_skills"}
	for _, tbl := range tables {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}

	// Running Open again on the same file should be idempotent (migrations are IF NOT EXISTS)
	db2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	db2.Close()

	os.Remove(path)
}

func TestOpenInMemory(t *testing.T) {
	d, err := Open("file:testopen_inmem?mode=memory&cache=shared&_foreign_keys=on")
	if err != nil {
		t.Fatalf("Open memory: %v", err)
	}
	defer d.Close()
	if d == nil {
		t.Fatal("expected non-nil db")
	}
}
