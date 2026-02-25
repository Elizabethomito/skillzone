// Package handlers contains the HTTP handler logic for the Skillzone API.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — package structure
// ────────────────────────────────────────────────────────────────────
// All handler files share the same "handlers" package so they can call
// each other's helpers freely without exporting them. The files are
// split by domain (auth, events, skills, sync) purely for readability.
//
// The central type is Server. It holds the two things every handler
// needs: a database connection and the JWT secret. Putting shared
// dependencies on a struct (instead of global variables) makes the code
// easier to test — each test creates its own Server with its own
// in-memory database and no test pollutes another.
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// respond writes v as JSON with the given HTTP status code.
// Setting Content-Type before WriteHeader is important — once
// WriteHeader is called the headers are flushed and cannot be changed.
func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Ignoring the encode error: if the client disconnected mid-write
	// there is nothing useful we can do, and logging every dropped
	// connection would be very noisy.
	_ = json.NewEncoder(w).Encode(body)
}

// respondError is a convenience wrapper that sends a JSON object with
// a single "error" key, e.g. {"error": "event not found"}.
func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

// decode reads and parses a JSON request body into v.
// It uses a streaming decoder (json.NewDecoder) which is more memory-
// efficient than ioutil.ReadAll + json.Unmarshal for large bodies.
func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Server holds shared dependencies for all handlers.
// Using a struct instead of package-level globals means tests can spin
// up many independent Server instances without state leaking between them.
type Server struct {
	// DB is the SQLite connection pool. database/sql is safe for
	// concurrent use — the pool manages multiple connections internally.
	DB *sql.DB
	// Secret is the HMAC key used to sign and verify JWTs.
	Secret string
}
