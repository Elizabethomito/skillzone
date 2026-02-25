// Package db handles SQLite initialisation and schema migrations.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — why modernc.org/sqlite instead of go-sqlite3?
// ────────────────────────────────────────────────────────────────────
// go-sqlite3 is a CGo binding — it compiles C code alongside your Go
// code. This requires a C compiler (gcc/clang) to be present on the
// build machine and produces a binary that depends on the system's C
// runtime. On many deployment targets (scratch Docker images, some CI
// pipelines, Windows without MinGW) this causes hard-to-debug errors.
//
// modernc.org/sqlite is a pure-Go port of SQLite — no C compiler
// needed, no CGo, cross-compiles cleanly. The tradeoff is a slightly
// larger binary and marginally slower throughput, neither of which
// matters for this project.
//
// The only visible difference: the driver name changes from "sqlite3"
// to "sqlite".
package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	// Blank import: the modernc driver registers itself with
	// database/sql under the name "sqlite" when this package loads.
	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite database at dsn and runs all migrations.
//
// Recommended DSN formats for modernc.org/sqlite:
//   - Production file: "skillzone.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
//   - Tests:           "file:testXYZ?mode=memory&cache=shared&_foreign_keys=on"
//
// LEARNING NOTE — DSN (Data Source Name)
// A DSN is just a connection string. For SQLite it's the file path plus
// optional URI query parameters that configure pragma settings. Using
// URI parameters means every connection from the pool gets the pragmas
// applied automatically — important because database/sql can open many
// connections and each one starts with SQLite defaults.
func Open(dsn string) (*sql.DB, error) {
	// sql.Open does NOT open a real connection yet — it just validates
	// the driver name and stores the DSN. The first real connection is
	// made lazily on the first query (or explicitly via db.Ping()).
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Run all CREATE TABLE IF NOT EXISTS statements.
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Println("database ready:", dsn)
	return db, nil
}

// migrate runs each DDL statement in the schema individually.
//
// LEARNING NOTE — why not one big Exec(schema)?
// The go-sqlite3 and modernc drivers both execute only the FIRST
// statement when you pass a multi-statement string to Exec(). To run
// all of them we split on ";" and loop. Empty strings (whitespace
// between statements) are skipped.
func migrate(db *sql.DB) error {
	stmts := strings.Split(schema, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration statement failed: %w\nstatement: %s", err, stmt)
		}
	}
	return nil
}

// schema contains every CREATE TABLE statement for the application.
//
// LEARNING NOTE — schema design choices
//
//	users          — single table for both students and companies; the
//	                 "role" column distinguishes them. Simpler than two
//	                 separate tables for a project this size.
//
//	skills         — a global catalogue of skill badges. Companies pick
//	                 from here when creating events.
//
//	events         — hosted by a company. Stores a check_in_code (a
//	                 random UUID) that is the shared secret embedded in
//	                 the QR code shown at check-in.
//
//	event_skills   — many-to-many join: one event can award many skills.
//
//	registrations  — a student's intent to attend an event. Created
//	                 online; the UNIQUE constraint prevents duplicates.
//
//	attendances    — the offline check-in proof. The student stores this
//	                 locally and syncs it later. UNIQUE(event_id,student_id)
//	                 means the upsert in SyncAttendance is safe to retry.
//
//	user_skills    — the awarded badge. Written when attendance is verified.
//	                 UNIQUE(user_id,skill_id,event_id) makes award idempotent.
const schema = `
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL CHECK(role IN ('student','company')),
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS skills (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS events (
    id            TEXT PRIMARY KEY,
    host_id       TEXT NOT NULL REFERENCES users(id),
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    location      TEXT NOT NULL DEFAULT '',
    start_time    DATETIME NOT NULL,
    end_time      DATETIME NOT NULL,
    status        TEXT NOT NULL DEFAULT 'upcoming'
                      CHECK(status IN ('upcoming','active','completed')),
    check_in_code TEXT NOT NULL DEFAULT '',
    capacity         INTEGER,
    slots_remaining  INTEGER,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS event_skills (
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (event_id, skill_id)
);

CREATE TABLE IF NOT EXISTS registrations (
    id            TEXT PRIMARY KEY,
    event_id      TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    student_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status        TEXT NOT NULL DEFAULT 'confirmed'
                      CHECK(status IN ('confirmed','conflict_pending','waitlisted')),
    UNIQUE (event_id, student_id)
);

CREATE TABLE IF NOT EXISTS attendances (
    id         TEXT PRIMARY KEY,
    event_id   TEXT NOT NULL REFERENCES events(id),
    student_id TEXT NOT NULL REFERENCES users(id),
    payload    TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending'
                   CHECK(status IN ('pending','verified','rejected')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, student_id)
);

CREATE TABLE IF NOT EXISTS user_skills (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_id   TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    event_id   TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    awarded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, skill_id, event_id)
);
`
