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
	// Configuration from environment variables with sensible defaults.
	dsn := getenv("DATABASE_URL", "skillzone.db?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	jwtSecret := getenv("JWT_SECRET", "changeme-use-a-real-secret-in-production")
	addr := getenv("ADDR", ":8080")

	database, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	srv := &handlers.Server{
		DB:     database,
		Secret: jwtSecret,
	}

	mux := http.NewServeMux()

	// ---- Public routes ----
	mux.HandleFunc("POST /api/auth/register", srv.Register)
	mux.HandleFunc("POST /api/auth/login", srv.Login)
	mux.HandleFunc("GET /api/events", srv.ListEvents)
	mux.HandleFunc("GET /api/events/{id}", srv.GetEvent)
	mux.HandleFunc("GET /api/skills", srv.ListSkills)

	// ---- Authenticated routes ----
	auth := middleware.Authenticate(jwtSecret)
	onlyCompany := middleware.RequireRole("company")
	onlyStudent := middleware.RequireRole("student")

	mux.Handle("GET /api/auth/me",
		auth(http.HandlerFunc(srv.Me)))

	// Company-only
	mux.Handle("POST /api/events",
		auth(onlyCompany(http.HandlerFunc(srv.CreateEvent))))
	mux.Handle("GET /api/events/{id}/checkin-code",
		auth(onlyCompany(http.HandlerFunc(srv.GetEventCheckInCode))))
	mux.Handle("POST /api/skills",
		auth(onlyCompany(http.HandlerFunc(srv.CreateSkill))))

	// Student-only
	mux.Handle("POST /api/events/{id}/register",
		auth(onlyStudent(http.HandlerFunc(srv.RegisterForEvent))))
	mux.Handle("POST /api/sync/attendance",
		auth(onlyStudent(http.HandlerFunc(srv.SyncAttendance))))
	mux.Handle("GET /api/users/me/skills",
		auth(onlyStudent(http.HandlerFunc(srv.GetMySkills))))
	mux.Handle("GET /api/users/me/registrations",
		auth(onlyStudent(http.HandlerFunc(srv.GetMyRegistrations))))

	// Wrap everything in CORS middleware
	handler := middleware.CORS(mux)

	log.Printf("Skillzone API listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
