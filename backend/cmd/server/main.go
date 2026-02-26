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
//
// LOGGING
// We use log/slog (stdlib, Go 1.21+) as the structured logger.
// In a terminal tint wraps it with ANSI colour codes so each log level
// gets a distinct colour (DEBUG=grey, INFO=green, WARN=yellow, ERROR=red).
// tint.NewHandler detects whether stdout is a real TTY; colour is
// automatically suppressed when the output is piped or redirected so log
// files stay clean.
//
// GRACEFUL SHUTDOWN
// http.Server.Shutdown drains in-flight requests before closing.  We
// listen for SIGINT / SIGTERM on a channel, trigger Shutdown in a
// goroutine, and wait for it to finish before main() returns.  This
// means the database deferred close always runs and no request is cut
// off mid-flight.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"

	"github.com/Elizabethomito/skillzone/backend/internal/db"
	"github.com/Elizabethomito/skillzone/backend/internal/handlers"
	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
)

func main() {
	// ── Logger ───────────────────────────────────────────────────────
	// tint.NewHandler wraps slog with ANSI colour codes when stdout is a
	// real TTY; it falls back to plain text when piped / redirected so
	// log files stay machine-readable.
	//   DEBUG → grey    INFO → green    WARN → yellow    ERROR → red
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen, // e.g. "3:04PM" — compact for a dev terminal
		NoColor:    !isatty(os.Stdout),
	}))
	// Replace the global logger so any package that calls slog.Info(…)
	// also uses the coloured handler.
	slog.SetDefault(logger)

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
		slog.Error("open database", "err", err)
		os.Exit(1)
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
	mux.Handle("PUT /api/events/{id}",
		auth(onlyCompany(http.HandlerFunc(srv.UpdateEvent))))
	mux.Handle("GET /api/events/{id}/checkin-code",
		auth(onlyCompany(http.HandlerFunc(srv.GetEventCheckInCode))))
	mux.Handle("PATCH /api/events/{id}/status",
		auth(onlyCompany(http.HandlerFunc(srv.UpdateEventStatus))))
	mux.Handle("GET /api/events/{id}/registrations",
		auth(onlyCompany(http.HandlerFunc(srv.GetEventRegistrations))))
	mux.Handle("PATCH /api/events/{id}/registrations/{reg_id}",
		auth(onlyCompany(http.HandlerFunc(srv.ResolveRegistrationConflict))))
	mux.Handle("DELETE /api/events/{id}/registrations/{reg_id}",
		auth(onlyCompany(http.HandlerFunc(srv.KickRegistration))))
	mux.Handle("POST /api/skills",
		auth(onlyCompany(http.HandlerFunc(srv.CreateSkill))))
	mux.Handle("GET /api/users/students",
		auth(onlyCompany(http.HandlerFunc(srv.SearchStudents))))

	// Student-only routes.
	mux.Handle("POST /api/events/{id}/register",
		auth(onlyStudent(http.HandlerFunc(srv.RegisterForEvent))))
	mux.Handle("DELETE /api/events/{id}/register",
		auth(onlyStudent(http.HandlerFunc(srv.UnregisterFromEvent))))
	// ↓ Core local-first sync endpoint — see handlers/sync.go
	mux.Handle("POST /api/sync/attendance",
		auth(onlyStudent(http.HandlerFunc(srv.SyncAttendance))))
	mux.Handle("GET /api/users/me/skills",
		auth(onlyStudent(http.HandlerFunc(srv.GetMySkills))))
	mux.Handle("GET /api/users/me/registrations",
		auth(onlyStudent(http.HandlerFunc(srv.GetMyRegistrations))))

	// Wrap the entire mux in CORS and the request logger so every
	// request is printed: method, path, status, latency.
	handler := middleware.CORS(requestLogger(mux))

	// ── HTTP server ───────────────────────────────────────────────────
	// Populate http.Server explicitly so we can call Shutdown() on it
	// during graceful shutdown.  The timeouts protect against slow
	// clients hogging connections in production.
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────
	// We start the server in a goroutine so main() can block on the
	// signal channel.  When SIGINT or SIGTERM arrives we give in-flight
	// requests up to 10 s to finish before forcefully closing.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("Skillzone API started", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Block until a signal is received.
	sig := <-quit
	slog.Info("shutdown signal received", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	} else {
		slog.Info("server stopped cleanly")
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

// isatty reports whether f is connected to an interactive terminal.
// Used to decide whether to emit ANSI colour codes — we skip them when
// stdout is a pipe or a file so logs stay clean for tools like grep.
func isatty(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	// A character device whose name contains "tty" or whose mode has
	// ModeCharDevice set is an interactive terminal.
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// responseWriter wraps http.ResponseWriter to capture the status code
// written by a handler so the request logger can record it.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// requestLogger is an HTTP middleware that logs every request using
// the global slog logger.  It records:
//   - HTTP method and path
//   - Response status code (colour-coded by tint based on level)
//   - Latency (wall-clock time the handler took)
//
// 2xx/3xx → INFO   4xx → WARN   5xx → ERROR
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		latency := time.Since(start)

		level := slog.LevelInfo
		switch {
		case rw.status >= 500:
			level = slog.LevelError
		case rw.status >= 400:
			level = slog.LevelWarn
		}

		slog.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"latency", latency,
		)
	})
}
