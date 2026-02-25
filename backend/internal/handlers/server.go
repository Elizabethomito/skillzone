// Package handlers contains the HTTP handler logic for the Skillzone API.
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

// respond writes a JSON body with the given status code.
func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// respondError writes a JSON error message.
func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

// decode reads JSON from the request body into v.
func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Server holds shared dependencies for all handlers.
type Server struct {
	DB     *sql.DB
	Secret string
}
