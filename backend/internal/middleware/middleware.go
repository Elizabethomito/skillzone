// Package middleware provides HTTP middleware for the Skillzone server.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
)

type contextKey string

const (
	ContextUserID contextKey = "user_id"
	ContextRole   contextKey = "role"
)

// Authenticate validates a Bearer JWT from the Authorization header.
// On success the user_id and role are stored in the request context.
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

			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole rejects requests whose context role does not match any of the
// allowed roles.  Must be used after Authenticate.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
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

// CORS adds permissive CORS headers so the React PWA can call the API.
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
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(ContextUserID).(string)
	return id
}

// GetRole retrieves the authenticated user's role from the context.
func GetRole(ctx context.Context) string {
	role, _ := ctx.Value(ContextRole).(string)
	return role
}
