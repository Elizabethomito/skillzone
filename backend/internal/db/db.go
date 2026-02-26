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

// migrate runs each DDL statement in the schema individually, then applies
// any incremental alter-table migrations that cannot be expressed as
// CREATE TABLE IF NOT EXISTS statements.
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

	// ── Incremental migrations ────────────────────────────────────────────────
	// These handle schema changes that cannot be applied via CREATE TABLE IF NOT
	// EXISTS (e.g. adding a new column to an existing table, or widening a CHECK
	// constraint in SQLite which requires a table rebuild).
	if err := applyIncrementalMigrations(db); err != nil {
		return fmt.Errorf("incremental migrations: %w", err)
	}

	return nil
}

// applyIncrementalMigrations handles schema changes on existing databases.
// Each migration is idempotent — it checks whether the change already exists
// before attempting it so the function is safe to run on every startup.
func applyIncrementalMigrations(db *sql.DB) error {
	// ── Migration 1: add kicked_at column to registrations ────────────────
	// Check whether the column already exists.
	rows, err := db.Query(`PRAGMA table_info(registrations)`)
	if err != nil {
		return fmt.Errorf("PRAGMA table_info: %w", err)
	}
	hasKickedAt := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, colType string
		var dfltValue sql.NullString
		if scanErr := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); scanErr == nil {
			if name == "kicked_at" {
				hasKickedAt = true
				break
			}
		}
	}
	rows.Close()

	if !hasKickedAt {
		if _, err := db.Exec(`ALTER TABLE registrations ADD COLUMN kicked_at DATETIME`); err != nil {
			return fmt.Errorf("add kicked_at column: %w", err)
		}
		log.Println("migration: added registrations.kicked_at")
	}

	// ── Migration 2: widen registrations CHECK constraint to include 'rejected'
	// SQLite cannot ALTER a CHECK constraint directly — we must recreate the
	// table.  We detect whether the old constraint is still in place by
	// attempting an INSERT with the 'rejected' status inside a savepoint,
	// then rolling back.
	needsConstraintFix := false
	_, testErr := db.Exec(`SAVEPOINT _check_rejected_test`)
	if testErr == nil {
		_, insertErr := db.Exec(
			`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
			 VALUES ('__test__', '__test__', '__test__', CURRENT_TIMESTAMP, 'rejected')`,
		)
		db.Exec(`ROLLBACK TO SAVEPOINT _check_rejected_test`) //nolint:errcheck
		db.Exec(`RELEASE SAVEPOINT _check_rejected_test`)     //nolint:errcheck
		if insertErr != nil {
			needsConstraintFix = true
		}
	}

	if needsConstraintFix {
		log.Println("migration: widening registrations CHECK constraint to include 'rejected'")
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for constraint migration: %w", err)
		}
		defer tx.Rollback() //nolint:errcheck

		// Recreate the table with the new constraint.
		stmts := []string{
			`CREATE TABLE registrations_new (
				id            TEXT PRIMARY KEY,
				event_id      TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
				student_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				status        TEXT NOT NULL DEFAULT 'confirmed'
				                  CHECK(status IN ('confirmed','conflict_pending','waitlisted','rejected')),
				kicked_at     DATETIME,
				UNIQUE (event_id, student_id)
			)`,
			`INSERT INTO registrations_new SELECT id, event_id, student_id, registered_at, status, kicked_at FROM registrations`,
			`DROP TABLE registrations`,
			`ALTER TABLE registrations_new RENAME TO registrations`,
		}
		for _, s := range stmts {
			if _, err := tx.Exec(s); err != nil {
				return fmt.Errorf("constraint migration step failed: %w\nstmt: %s", err, s)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit constraint migration: %w", err)
		}
		log.Println("migration: registrations table rebuilt with new CHECK constraint")
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
                      CHECK(status IN ('confirmed','conflict_pending','waitlisted','rejected')),
    kicked_at     DATETIME,
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
