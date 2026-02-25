package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
)

// CreateSkill handles POST /api/skills  (company only)
func (s *Server) CreateSkill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	skill := models.Skill{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now().UTC(),
	}

	_, err := s.DB.ExecContext(r.Context(),
		`INSERT INTO skills (id, name, description, created_at) VALUES (?, ?, ?, ?)`,
		skill.ID, skill.Name, skill.Description, skill.CreatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			respondError(w, http.StatusConflict, "skill already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, "could not create skill")
		return
	}

	respond(w, http.StatusCreated, skill)
}

// ListSkills handles GET /api/skills
func (s *Server) ListSkills(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT id, name, description, created_at FROM skills ORDER BY name ASC`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	skills := []models.Skill{}
	for rows.Next() {
		var sk models.Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.CreatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "scan error")
			return
		}
		skills = append(skills, sk)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "rows error")
		return
	}

	respond(w, http.StatusOK, skills)
}
