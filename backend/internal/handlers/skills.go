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

// SearchStudents handles GET /api/users/students?skill_id=<uuid>  (company only)
//
// Returns students who possess specific verified skill badges. Companies use
// this as a "talent filter" — e.g. "show me all students with Python + Cloud badges".
// Multiple skill_id query params are ANDed: the student must hold ALL listed skills.
// Omitting skill_id returns all students on the platform.
func (s *Server) SearchStudents(w http.ResponseWriter, r *http.Request) {
	skillIDs := r.URL.Query()["skill_id"]

	type StudentWithSkills struct {
		models.User
		Skills []models.UserSkill `json:"skills"`
	}

	var rows interface{ Next() bool; Scan(...interface{}) error; Close() error; Err() error }
	var err error

	if len(skillIDs) == 0 {
		// Return all students.
		rows, err = s.DB.QueryContext(r.Context(),
			`SELECT id, email, name, role, created_at, updated_at
			 FROM users WHERE role = 'student' ORDER BY name ASC`)
	} else {
		// Only students who hold ALL requested skills.
		placeholders := strings.Repeat("?,", len(skillIDs))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]interface{}, len(skillIDs)+1)
		args[0] = len(skillIDs)
		for i, id := range skillIDs {
			args[i+1] = id
		}
		rows, err = s.DB.QueryContext(r.Context(),
			`SELECT u.id, u.email, u.name, u.role, u.created_at, u.updated_at
			 FROM users u
			 WHERE u.role = 'student'
			   AND (SELECT COUNT(DISTINCT skill_id) FROM user_skills
			        WHERE user_id = u.id AND skill_id IN (`+placeholders+`)) = ?
			 ORDER BY u.name ASC`,
			append(skillIDsToInterfaces(skillIDs), args[0])...,
		)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var students []StudentWithSkills
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "scan error")
			return
		}
		// Fetch this student's skills.
		skillRows, err := s.DB.QueryContext(r.Context(),
			`SELECT us.id, us.user_id, us.skill_id, us.event_id, us.awarded_at,
			        sk.id, sk.name, sk.description, sk.created_at
			 FROM user_skills us
			 JOIN skills sk ON sk.id = us.skill_id
			 WHERE us.user_id = ?
			 ORDER BY us.awarded_at DESC`, u.ID)
		if err == nil {
			defer skillRows.Close()
			var skills []models.UserSkill
			for skillRows.Next() {
				var us models.UserSkill
				us.Skill = &models.Skill{}
				_ = skillRows.Scan(&us.ID, &us.UserID, &us.SkillID, &us.EventID, &us.AwardedAt,
					&us.Skill.ID, &us.Skill.Name, &us.Skill.Description, &us.Skill.CreatedAt)
				skills = append(skills, us)
			}
			if skills == nil {
				skills = []models.UserSkill{}
			}
			students = append(students, StudentWithSkills{User: u, Skills: skills})
		} else {
			students = append(students, StudentWithSkills{User: u, Skills: []models.UserSkill{}})
		}
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "rows error")
		return
	}
	if students == nil {
		students = []StudentWithSkills{}
	}
	respond(w, http.StatusOK, students)
}

// skillIDsToInterfaces converts []string to []interface{} for variadic SQL args.
func skillIDsToInterfaces(ids []string) []interface{} {
	out := make([]interface{}, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}
