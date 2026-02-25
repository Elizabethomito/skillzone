package handlers

import (
"net/http"
"strings"
"time"

"github.com/Elizabethomito/skillzone/backend/internal/models"
"github.com/google/uuid"
)

// CreateSkill handles POST /api/skills  (company only)
//
// Skills are global — any company can create them and any event can link to them.
// The UNIQUE constraint on skills.name (defined in the schema) prevents
// duplicate badge names; the handler surfaces that as a 409 Conflict.
func (s *Server) CreateSkill(w http.ResponseWriter, r *http.Request) {
// Anonymous struct for the request body — fine for small, one-off shapes
// that are only used in this function. For types used in multiple places
// we define them in the models package.
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
// Public — no authentication required. Skills are the catalogue of badges
// the platform offers; anyone browsing events needs to see them.
func (s *Server) ListSkills(w http.ResponseWriter, r *http.Request) {
rows, err := s.DB.QueryContext(r.Context(),
`SELECT id, name, description, created_at FROM skills ORDER BY name ASC`)
if err != nil {
respondError(w, http.StatusInternalServerError, "database error")
return
}
defer rows.Close()

// Initialised to empty slice so JSON encodes as [] not null when empty.
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
