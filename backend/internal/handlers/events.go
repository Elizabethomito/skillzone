package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
)

// CreateEvent handles POST /api/events  (company only)
func (s *Server) CreateEvent(w http.ResponseWriter, r *http.Request) {
	hostID := middleware.GetUserID(r.Context())

	var req models.CreateEventRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		respondError(w, http.StatusBadRequest, "start_time and end_time are required")
		return
	}
	if !req.EndTime.After(req.StartTime) {
		respondError(w, http.StatusBadRequest, "end_time must be after start_time")
		return
	}

	event := models.Event{
		ID:          uuid.NewString(),
		HostID:      hostID,
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      models.EventStatusUpcoming,
		CheckInCode: uuid.NewString(), // random secret embedded in QR code
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(r.Context(),
		`INSERT INTO events (id, host_id, title, description, location, start_time, end_time, status, check_in_code, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.HostID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.Status, event.CheckInCode,
		event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not create event")
		return
	}

	// Link skills to the event
	for _, skillID := range req.SkillIDs {
		_, err = tx.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO event_skills (event_id, skill_id) VALUES (?, ?)`,
			event.ID, skillID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "could not link skill")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	event.Skills = s.fetchEventSkills(r, event.ID)
	respond(w, http.StatusCreated, event)
}

// ListEvents handles GET /api/events
func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT id, host_id, title, description, location, start_time, end_time, status, created_at, updated_at
		 FROM events ORDER BY start_time ASC`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	events := []models.Event{}
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.HostID, &e.Title, &e.Description, &e.Location,
			&e.StartTime, &e.EndTime, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "scan error")
			return
		}
		e.Skills = s.fetchEventSkills(r, e.ID)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "rows error")
		return
	}

	respond(w, http.StatusOK, events)
}

// GetEvent handles GET /api/events/{id}
func (s *Server) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var e models.Event
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT id, host_id, title, description, location, start_time, end_time, status, created_at, updated_at
		 FROM events WHERE id = ?`, id,
	).Scan(&e.ID, &e.HostID, &e.Title, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "event not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	e.Skills = s.fetchEventSkills(r, e.ID)
	respond(w, http.StatusOK, e)
}

// GetEventCheckInCode handles GET /api/events/{id}/checkin-code  (host only)
// Returns the check-in code used to generate the QR payload.
func (s *Server) GetEventCheckInCode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	hostID := middleware.GetUserID(r.Context())

	var dbHostID, checkInCode string
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT host_id, check_in_code FROM events WHERE id = ?`, id,
	).Scan(&dbHostID, &checkInCode)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "event not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if dbHostID != hostID {
		respondError(w, http.StatusForbidden, "you are not the host of this event")
		return
	}

	respond(w, http.StatusOK, map[string]string{
		"event_id":      id,
		"check_in_code": checkInCode,
	})
}

// RegisterForEvent handles POST /api/events/{id}/register  (student only)
func (s *Server) RegisterForEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	studentID := middleware.GetUserID(r.Context())

	// Check event exists
	var exists bool
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM events WHERE id = ?)`, eventID,
	).Scan(&exists)
	if err != nil || !exists {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	reg := models.Registration{
		ID:           uuid.NewString(),
		EventID:      eventID,
		StudentID:    studentID,
		RegisteredAt: time.Now().UTC(),
	}

	_, err = s.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at)
		 VALUES (?, ?, ?, ?)`,
		reg.ID, reg.EventID, reg.StudentID, reg.RegisteredAt,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not register")
		return
	}

	respond(w, http.StatusCreated, reg)
}

// fetchEventSkills is a helper to load skill badges for an event.
func (s *Server) fetchEventSkills(r *http.Request, eventID string) []models.Skill {
	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT sk.id, sk.name, sk.description, sk.created_at
		 FROM skills sk
		 JOIN event_skills es ON es.skill_id = sk.id
		 WHERE es.event_id = ?`, eventID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var skills []models.Skill
	for rows.Next() {
		var sk models.Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.CreatedAt); err == nil {
			skills = append(skills, sk)
		}
	}
	return skills
}
