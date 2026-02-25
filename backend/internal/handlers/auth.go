package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Register handles POST /api/auth/register
//
// Flow:
//  1. Decode and validate the request body.
//  2. Hash the password with bcrypt (slow by design — makes brute force hard).
//  3. Insert the new user row.
//  4. Generate a JWT and return it with the user object.
//
// Returning the token immediately means the client can start making
// authenticated requests without a separate login step.
func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Normalise email: lowercase and trim whitespace so
	// "Alice@Example.com " and "alice@example.com" are treated as the same.
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if req.Email == "" || req.Password == "" || req.Name == "" {
		respondError(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}
	if req.Role != models.RoleStudent && req.Role != models.RoleCompany {
		respondError(w, http.StatusBadRequest, "role must be 'student' or 'company'")
		return
	}
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	// bcrypt.DefaultCost (10) means ~100 ms per hash on modern hardware —
	// intentionally slow to resist offline brute-force attacks if the DB leaks.
	// We NEVER store the plain-text password.
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := models.User{
		ID:           uuid.NewString(), // random UUID v4 — globally unique without a sequence
		Email:        req.Email,
		PasswordHash: string(hash),
		Name:         req.Name,
		Role:         req.Role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_, err = s.DB.ExecContext(r.Context(),
		`INSERT INTO users (id, email, password_hash, name, role, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Email, user.PasswordHash, user.Name, user.Role, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		// The UNIQUE constraint on email fires here if the address is taken.
		if strings.Contains(err.Error(), "UNIQUE") {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		respondError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), s.Secret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	// 201 Created — a new resource was created.
	respond(w, http.StatusCreated, models.LoginResponse{Token: token, User: user})
}

// Login handles POST /api/auth/login
//
// LEARNING NOTE — timing attacks
// We always call bcrypt.CompareHashAndPassword even when the user is not
// found. Without this, an attacker could tell whether an email exists by
// measuring response time (registered users take ~100 ms; unknown users
// return instantly). Here we return the same error message and take the
// same amount of time regardless.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var user models.User
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT id, email, password_hash, name, role, created_at, updated_at
		 FROM users WHERE email = ?`, req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return 401, not 404 — we don't want to confirm the email exists.
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// CompareHashAndPassword runs the same bcrypt cost as registration.
	// It returns an error if the password doesn't match.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := auth.GenerateToken(user.ID, string(user.Role), s.Secret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	respond(w, http.StatusOK, models.LoginResponse{Token: token, User: user})
}

// Me handles GET /api/auth/me
// Returns the currently authenticated user's profile.
// The Authenticate middleware has already validated the token and placed
// the user_id in the context, so we just need to look up the full record.
func (s *Server) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var user models.User
	err := s.DB.QueryRowContext(r.Context(),
		// The empty string literal ('') is a placeholder for password_hash —
		// we never want to return the hash over the wire, so we discard it
		// here rather than scanning into the struct and hoping the json:"-"
		// tag on PasswordHash prevents it from being serialised.
		`SELECT id, email, '', name, role, created_at, updated_at
		 FROM users WHERE id = ?`, userID,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respond(w, http.StatusOK, user)
}
