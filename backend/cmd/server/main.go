// main is the entry point for the Skillzone API server.
//
// It reads configuration from environment variables, opens the SQLite
// database, registers all HTTP routes, and starts listening.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — how this file fits into the project
// ────────────────────────────────────────────────────────────────────
// This file is the "composition root" — the single place where all the
// independent packages (db, handlers, middleware) are wired together.
// Keeping this wiring in main.go means every other package stays easy
// to test in isolation (they never import each other in a circle).
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Elizabethomito/skillzone/backend/internal/db"
	"github.com/Elizabethomito/skillzone/backend/internal/handlers"
	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
)

func main() {
	// ── Configuration ────────────────────────────────────────────────
	// Read settings from the environment so the same binary can be used
	// in development, CI, and production without recompiling.
	//
	// DATABASE_URL uses modernc.org/sqlite URI parameters:
	//   _pragma=foreign_keys(1)  — enforce FK constraints on every connection
	//   _pragma=journal_mode(WAL) — Write-Ahead Logging: readers don't block writers
	//   _pragma=busy_timeout(5000) — wait up to 5 s instead of returning SQLITE_BUSY
	dsn := getenv("DATABASE_URL",
		"skillzone.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	jwtSecret := getenv("JWT_SECRET", "changeme-use-a-real-secret-in-production")
	addr := getenv("ADDR", ":8080")

	// ── Database ─────────────────────────────────────────────────────
	// db.Open creates the file if it doesn't exist and runs all CREATE
	// TABLE IF NOT EXISTS migrations automatically.
	database, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	// ── Handlers ─────────────────────────────────────────────────────
	// Server is a plain struct that holds the two shared dependencies
	// (database handle and JWT secret). All handler methods live on it.
	srv := &handlers.Server{
		DB:     database,
		Secret: jwtSecret,
	}

	// ── Router ───────────────────────────────────────────────────────
	// Go 1.22+ ServeMux supports method prefixes ("GET /path") and path
	// wildcards ("{id}") natively — no third-party router needed.
	mux := http.NewServeMux()

	// Public routes — no token required.
	mux.HandleFunc("POST /api/auth/register", srv.Register)
	mux.HandleFunc("POST /api/auth/login", srv.Login)
	mux.HandleFunc("GET /api/events", srv.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", srv.GetEvent)
	mux.HandleFunc("GET /api/skills", srv.ListSkills)
	// Demo seed — loads all fixture data; safe to call multiple times (idempotent).
	// Remove or gate behind an env flag before any real deployment.
	mux.HandleFunc("POST /api/admin/seed", srv.SeedDemo)

	// ── Middleware helpers ────────────────────────────────────────────
	// middleware.Authenticate returns a function that wraps any handler.
	// middleware.RequireRole does the same but also checks the role claim.
	// Chaining them: auth(onlyCompany(handler)) means:
	//   1. Authenticate runs first  → sets user_id/role in context
	//   2. RequireRole runs second  → allows or rejects based on role
	//   3. handler runs last        → does the actual work
	auth := middleware.Authenticate(jwtSecret)
	onlyCompany := middleware.RequireRole("company")
	onlyStudent := middleware.RequireRole("student")

	// Authenticated — any logged-in user.
	mux.Handle("GET /api/auth/me",
		auth(http.HandlerFunc(srv.Me)))

	// Company-only routes.
	mux.Handle("POST /api/events",
		auth(onlyCompany(http.HandlerFunc(srv.CreateEvent))))
	mux.Handle("GET /api/events/{id}/checkin-code",
		auth(onlyCompany(http.HandlerFunc(srv.GetEventCheckInCode))))
	mux.Handle("PATCH /api/events/{id}/status",
		auth(onlyCompany(http.HandlerFunc(srv.UpdateEventStatus))))
	mux.Handle("GET /api/events/{id}/registrations",
		auth(onlyCompany(http.HandlerFunc(srv.GetEventRegistrations))))
	mux.Handle("PATCH /api/events/{id}/registrations/{reg_id}",
		auth(onlyCompany(http.HandlerFunc(srv.ResolveRegistrationConflict))))
	mux.Handle("POST /api/skills",
		auth(onlyCompany(http.HandlerFunc(srv.CreateSkill))))

	// Student-only routes.
	mux.Handle("POST /api/events/{id}/register",
		auth(onlyStudent(http.HandlerFunc(srv.RegisterForEvent))))
	// ↓ Core local-first sync endpoint — see handlers/sync.go
	mux.Handle("POST /api/sync/attendance",
		auth(onlyStudent(http.HandlerFunc(srv.SyncAttendance))))
	mux.Handle("GET /api/users/me/skills",
		auth(onlyStudent(http.HandlerFunc(srv.GetMySkills))))
	mux.Handle("GET /api/users/me/registrations",
		auth(onlyStudent(http.HandlerFunc(srv.GetMyRegistrations))))

	// Wrap the entire mux in CORS so browser PWA requests are allowed.
	handler := middleware.CORS(mux)

	log.Printf("Skillzone API listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// getenv returns the value of the named environment variable, or fallback
// if the variable is not set or is empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
