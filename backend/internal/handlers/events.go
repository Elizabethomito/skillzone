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
//
// LEARNING NOTE — database transactions
// We use a transaction (tx) here because we need to INSERT two things
// atomically: the event row AND the event_skills rows.
// If the server crashes after the event insert but before the skills
// insert, `defer tx.Rollback()` automatically undoes the event insert
// when the function returns — so we never leave the DB in a half-written state.
// `tx.Commit()` at the end makes both writes permanent at the same time.
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
// CheckInCode is a random UUID that acts as the shared secret for
// QR code verification. Only the host can retrieve it via
// GET /api/events/{id}/checkin-code.
CheckInCode: uuid.NewString(),
CreatedAt:   time.Now().UTC(),
UpdatedAt:   time.Now().UTC(),
}

tx, err := s.DB.BeginTx(r.Context(), nil)
if err != nil {
respondError(w, http.StatusInternalServerError, "database error")
return
}
defer tx.Rollback() //nolint:errcheck // Rollback is a no-op after Commit succeeds

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

// Link skills to the event.
// INSERT OR IGNORE means: if the (event_id, skill_id) pair already
// exists (the UNIQUE constraint fires), silently skip it rather than
// returning an error — safe for idempotent retries.
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
// Public endpoint — no authentication required.
// Note: check_in_code is NOT selected here; it must never be exposed publicly.
func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
rows, err := s.DB.QueryContext(r.Context(),
`SELECT id, host_id, title, description, location, start_time, end_time, status, created_at, updated_at
 FROM events ORDER BY start_time ASC`)
if err != nil {
respondError(w, http.StatusInternalServerError, "database error")
return
}
defer rows.Close()

// Initialise to an empty slice, not nil, so JSON encodes as [] not null.
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
// rows.Err() catches any error that occurred during iteration —
// it's separate from the error returned by rows.Next() reaching EOF.
if err := rows.Err(); err != nil {
respondError(w, http.StatusInternalServerError, "rows error")
return
}

respond(w, http.StatusOK, events)
}

// GetEvent handles GET /api/events/{id}
// r.PathValue("id") is the Go 1.22+ way to read path parameters from the
// standard library mux — no third-party router needed.
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
//
// This endpoint returns the check_in_code that goes into the host's QR code.
// Only the company that created the event can call this — enforced by
// comparing the JWT's user_id to the event's host_id.
// The Authenticate + RequireRole("company") middleware has already run.
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

// Extra ownership check — even a different company user cannot see
// another company's check-in code.
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
//
// INSERT OR IGNORE makes this idempotent: if the student presses "Register"
// twice (e.g. after a failed request), the second call silently succeeds
// without creating a duplicate row.
func (s *Server) RegisterForEvent(w http.ResponseWriter, r *http.Request) {
eventID := r.PathValue("id")
studentID := middleware.GetUserID(r.Context())

// Lightweight existence check before we try to insert.
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

// fetchEventSkills is an internal helper used by CreateEvent, ListEvents and
// GetEvent to load the skills attached to an event via the event_skills join table.
// We use a JOIN here: one SQL query instead of N queries (one per skill).
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
