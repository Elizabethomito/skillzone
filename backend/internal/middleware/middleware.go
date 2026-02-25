// Package middleware provides HTTP middleware for the Skillzone server.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — what is middleware?
// ────────────────────────────────────────────────────────────────────
// In HTTP servers, "middleware" is a function that wraps a handler to
// add behaviour before and/or after it runs. The pattern in Go is:
//
//   func MyMiddleware(next http.Handler) http.Handler {
//       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//           // do something before
//           next.ServeHTTP(w, r)  // call the real handler
//           // do something after
//       })
//   }
//
// Middleware can be chained: CORS(Authenticate(handler)) means CORS
// runs first, then Authenticate, then the handler.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
)

// contextKey is a private type for context keys in this package.
// Using a named type prevents key collisions with other packages that
// also store values in the request context.
type contextKey string

const (
	// ContextUserID is the key under which the authenticated user's ID
	// is stored in the request context after Authenticate runs.
	ContextUserID contextKey = "user_id"
	// ContextRole is the key for the user's role ("student"/"company").
	ContextRole contextKey = "role"
)

// Authenticate is a middleware factory — it returns a middleware function
// configured with the JWT secret. This lets us pass the secret once at
// startup rather than on every request.
//
// Flow:
//  1. Read the "Authorization: Bearer <token>" header.
//  2. Parse and validate the JWT.
//  3. Store user_id and role in the request context.
//  4. Call the next handler.
//
// If the token is missing or invalid, it responds with 401 and stops.
func Authenticate(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, `{"error":"missing or invalid Authorization header"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			claims, err := auth.ParseToken(tokenStr, secret)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Store the claims in the context so downstream handlers can
			// retrieve them without re-parsing the token.
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns a middleware that only allows requests whose context
// role matches one of the given roles. Must be used after Authenticate.
//
// Example: auth(RequireRole("company")(handler))
// means: authenticate first, then only let companies through.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	// Build a set for O(1) lookup — more efficient than scanning a slice
	// on every request, even though the list is tiny here.
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(ContextRole).(string)
			if !allowed[role] {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS adds permissive CORS headers so the React PWA can call the API
// from a different origin (e.g. localhost:5173 in dev, or a different
// port on the demo laptop).
//
// LEARNING NOTE — what is CORS?
// Browsers enforce the Same-Origin Policy: a page at origin A cannot
// fetch from origin B unless B explicitly allows it via CORS headers.
// "Access-Control-Allow-Origin: *" means any origin is allowed.
// The OPTIONS preflight is a browser pre-check; we must reply 204 so
// the real request is allowed to proceed.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUserID retrieves the authenticated user's ID from the context.
// Returns an empty string if Authenticate has not run (should not happen
// on protected routes, but defensive coding is good practice).
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(ContextUserID).(string)
	return id
}

// GetRole retrieves the authenticated user's role from the context.
func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(ContextRole).(string)
	return role
}
