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
func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := models.User{
		ID:           uuid.NewString(),
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

	respond(w, http.StatusCreated, models.LoginResponse{Token: token, User: user})
}

// Login handles POST /api/auth/login
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
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

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
func (s *Server) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var user models.User
	err := s.DB.QueryRowContext(r.Context(),
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
