// Package db handles SQLite initialisation and schema migrations.
package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens (or creates) the SQLite database at dsn and runs all migrations.
//
// Recommended DSNs:
//   - Production file:  "skillzone.db?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
//   - Tests:            "file:testXYZ?mode=memory&cache=shared&_foreign_keys=on"
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	log.Println("database ready:", dsn)
	return db, nil
}

// migrate runs each DDL statement individually so all tables are created.
// The go-sqlite3 driver only executes the first statement in a multi-statement
// Exec call, so we must split and run them one by one.
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
