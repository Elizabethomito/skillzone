package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
)

// CreateEvent handles POST /api/events  (company only)
//
// LEARNING NOTE — database transactions
// We use a transaction (tx) here because we need to INSERT two things
// atomically: the event row AND the event_skills rows.
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
		CheckInCode: uuid.NewString(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Set capacity if provided (> 0 means limited slots).
	if req.Capacity > 0 {
		c := req.Capacity
		event.Capacity = &c
		sr := req.Capacity
		event.SlotsRemaining = &sr
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(r.Context(),
		`INSERT INTO events (id, host_id, title, description, location, start_time, end_time, status, check_in_code, capacity, slots_remaining, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.HostID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.Status, event.CheckInCode,
		event.Capacity, event.SlotsRemaining,
		event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not create event")
		return
	}

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

// ListEvents handles GET /api/events (public)
func (s *Server) ListEvents(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT id, host_id, title, description, location, start_time, end_time, status,
        capacity, slots_remaining, created_at, updated_at
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
			&e.StartTime, &e.EndTime, &e.Status,
			&e.Capacity, &e.SlotsRemaining,
			&e.CreatedAt, &e.UpdatedAt); err != nil {
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

// GetEvent handles GET /api/events/{id} (public)
func (s *Server) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var e models.Event
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT id, host_id, title, description, location, start_time, end_time, status,
        capacity, slots_remaining, created_at, updated_at
 FROM events WHERE id = ?`, id,
	).Scan(&e.ID, &e.HostID, &e.Title, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.Status,
		&e.Capacity, &e.SlotsRemaining,
		&e.CreatedAt, &e.UpdatedAt)
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
// Returns a short-lived signed JWT (valid for auth.CheckInTokenDuration, currently
// 6 hours) that the host's PWA encodes into a QR code.
//
// The JWT carries event_id and host_sig claims and is signed with the server
// secret.  Students scan the QR and store the raw token string in IndexedDB.
// When they sync later the server verifies the signature to confirm the student
// was physically present during the QR's live window, without requiring an
// internet connection at scan time.
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

	// Generate a signed check-in token that expires after CheckInTokenDuration.
	// The token embeds event_id and host_sig so the sync handler can verify both
	// without any additional database lookup.
	token, err := auth.GenerateCheckInToken(id, checkInCode, s.Secret)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not generate check-in token")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"event_id":           id,
		"token":              token,
		"expires_in_seconds": int(auth.CheckInTokenDuration.Seconds()),
	})
}

// UpdateEventStatus handles PATCH /api/events/{id}/status  (host only)
//
// Allows the host to move an event through its lifecycle:
//
//	upcoming → active  (open check-in QR)
//	active   → completed  (end-of-day / early close)
//
// When an event is marked completed, the sync endpoint will still
// accept attendance records already in flight (within the 24-h window).
func (s *Server) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	hostID := middleware.GetUserID(r.Context())

	var req models.UpdateEventStatusRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	allowed := req.Status == models.EventStatusActive ||
		req.Status == models.EventStatusCompleted ||
		req.Status == models.EventStatusUpcoming
	if !allowed {
		respondError(w, http.StatusBadRequest, "status must be upcoming, active, or completed")
		return
	}

	// Verify ownership.
	var dbHostID string
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT host_id FROM events WHERE id = ?`, id,
	).Scan(&dbHostID)
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

	_, err = s.DB.ExecContext(r.Context(),
		`UPDATE events SET status = ?, updated_at = ? WHERE id = ?`,
		req.Status, time.Now().UTC(), id,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not update status")
		return
	}

	respond(w, http.StatusOK, map[string]string{"event_id": id, "status": string(req.Status)})
}

// GetEventRegistrations handles GET /api/events/{id}/registrations  (host only)
//
// Returns all registrations for the event, including student details and status.
// The host uses this to see the attendee list and to spot conflict_pending entries
// that need resolution (e.g. when two offline applicants both claim the last internship slot).
func (s *Server) GetEventRegistrations(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	hostID := middleware.GetUserID(r.Context())

	// Ownership check.
	var dbHostID string
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT host_id FROM events WHERE id = ?`, id,
	).Scan(&dbHostID)
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

	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT r.id, r.event_id, r.student_id, r.registered_at, r.status,
        u.name, u.email
 FROM registrations r
 JOIN users u ON u.id = r.student_id
 WHERE r.event_id = ?
 ORDER BY r.registered_at ASC`, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type RegWithStudent struct {
		models.Registration
		StudentName  string `json:"student_name"`
		StudentEmail string `json:"student_email"`
	}

	var regs []RegWithStudent
	for rows.Next() {
		var reg RegWithStudent
		if err := rows.Scan(
			&reg.ID, &reg.EventID, &reg.StudentID, &reg.RegisteredAt, &reg.Status,
			&reg.StudentName, &reg.StudentEmail,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "scan error")
			return
		}
		regs = append(regs, reg)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "rows error")
		return
	}

	if regs == nil {
		regs = []RegWithStudent{}
	}
	respond(w, http.StatusOK, regs)
}

// ResolveRegistrationConflict handles PATCH /api/events/{id}/registrations/{reg_id}  (host only)
//
// This is the conflict-resolution endpoint for the internship demo scenario.
//
// When two students both apply for the last slot while offline, both registrations
// arrive as conflict_pending. The host sees them in their dashboard and decides:
//   - "confirm"   → student gets the slot; slots_remaining is NOT further decremented
//     (it is already 0 from the first offline applicant that was auto-confirmed).
//   - "waitlist"  → student is placed on the waitlist; no slot change.
//
// LEARNING NOTE — why does this conflict arise?
// The server cannot know at sync time whether a human decision should override
// the "first-write-wins" rule for the last slot. We flag it as conflict_pending
// so the host can apply business logic (e.g. prefer the more qualified candidate).
func (s *Server) ResolveRegistrationConflict(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	regID := r.PathValue("reg_id")
	hostID := middleware.GetUserID(r.Context())

	var req models.ResolveConflictRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Action != "confirm" && req.Action != "waitlist" {
		respondError(w, http.StatusBadRequest, `action must be "confirm" or "waitlist"`)
		return
	}

	// Verify host owns the event.
	var dbHostID string
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT host_id FROM events WHERE id = ?`, eventID,
	).Scan(&dbHostID)
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

	// Check the registration exists and is conflict_pending.
	var currentStatus models.RegistrationStatus
	err = s.DB.QueryRowContext(r.Context(),
		`SELECT status FROM registrations WHERE id = ? AND event_id = ?`, regID, eventID,
	).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "registration not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if currentStatus != models.RegistrationConflictPending {
		respondError(w, http.StatusConflict, "registration is not in conflict_pending state")
		return
	}

	newStatus := models.RegistrationConfirmed
	if req.Action == "waitlist" {
		newStatus = models.RegistrationWaitlisted
	}

	_, err = s.DB.ExecContext(r.Context(),
		`UPDATE registrations SET status = ? WHERE id = ?`,
		newStatus, regID,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not update registration")
		return
	}

	respond(w, http.StatusOK, map[string]string{
		"registration_id": regID,
		"status":          string(newStatus),
	})
}

// RegisterForEvent handles POST /api/events/{id}/register  (student only)
//
// If the event has a capacity limit and slots are available, the registration
// is confirmed and slots_remaining is decremented atomically.
// If no slots remain, the registration is still recorded as conflict_pending
// so the host can resolve it manually.
func (s *Server) RegisterForEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	studentID := middleware.GetUserID(r.Context())

	// Load capacity info.
	var capacity, slotsRemaining sql.NullInt64
	var exists bool
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT 1, capacity, slots_remaining FROM events WHERE id = ?`, eventID,
	).Scan(&exists, &capacity, &slotsRemaining)
	if err != nil || !exists {
		respondError(w, http.StatusNotFound, "event not found")
		return
	}

	// Determine registration status.
	regStatus := models.RegistrationConfirmed
	if capacity.Valid && capacity.Int64 > 0 && slotsRemaining.Valid && slotsRemaining.Int64 <= 0 {
		regStatus = models.RegistrationConflictPending
	}

	reg := models.Registration{
		ID:           uuid.NewString(),
		EventID:      eventID,
		StudentID:    studentID,
		RegisteredAt: time.Now().UTC(),
		Status:       regStatus,
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	// INSERT OR IGNORE: if already registered, keep the existing record.
	result, err := tx.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
 VALUES (?, ?, ?, ?, ?)`,
		reg.ID, reg.EventID, reg.StudentID, reg.RegisteredAt, reg.Status,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not register")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 && regStatus == models.RegistrationConfirmed && capacity.Valid {
		// Decrement slot counter only when a new confirmed registration was inserted.
		_, err = tx.ExecContext(r.Context(),
			`UPDATE events SET slots_remaining = slots_remaining - 1, updated_at = ?
 WHERE id = ? AND slots_remaining > 0`,
			time.Now().UTC(), eventID,
		)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "could not update slots")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	respond(w, http.StatusCreated, reg)
}

// UpdateEvent handles PUT /api/events/{id}  (host only)
//
// Allows the event host to adjust title, description, location, start/end
// times, capacity, and linked skills. Partial updates are supported — omitted
// fields retain their current values. Capacity can be increased but not
// decreased below the number of confirmed registrations.
func (s *Server) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	hostID := middleware.GetUserID(r.Context())

	// Verify ownership and fetch current values.
	var e models.Event
	var cap, slots sql.NullInt64
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT id, host_id, title, description, location, start_time, end_time,
		        status, check_in_code, capacity, slots_remaining, created_at, updated_at
		 FROM events WHERE id = ?`, id,
	).Scan(&e.ID, &e.HostID, &e.Title, &e.Description, &e.Location,
		&e.StartTime, &e.EndTime, &e.Status, &e.CheckInCode,
		&cap, &slots, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "event not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if e.HostID != hostID {
		respondError(w, http.StatusForbidden, "you are not the host of this event")
		return
	}

	// Decode partial update request.
	var req models.UpdateEventRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Apply patch: only update fields that are provided.
	if t := strings.TrimSpace(req.Title); t != "" {
		e.Title = t
	}
	if req.Description != nil {
		e.Description = *req.Description
	}
	if req.Location != nil {
		e.Location = *req.Location
	}
	if req.StartTime != nil {
		e.StartTime = *req.StartTime
	}
	if req.EndTime != nil {
		e.EndTime = *req.EndTime
	}
	if !e.EndTime.After(e.StartTime) {
		respondError(w, http.StatusBadRequest, "end_time must be after start_time")
		return
	}
	now := time.Now().UTC()
	e.UpdatedAt = now

	// Capacity update: set or clear.
	newCap := cap
	newSlots := slots
	if req.Capacity != nil {
		if *req.Capacity <= 0 {
			// Clear capacity (unlimited)
			newCap = sql.NullInt64{}
			newSlots = sql.NullInt64{}
		} else {
			// Count confirmed registrations to prevent shrinking below current count.
			var confirmed int64
			_ = s.DB.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM registrations WHERE event_id = ? AND status = 'confirmed'`, id,
			).Scan(&confirmed)
			if int64(*req.Capacity) < confirmed {
				respondError(w, http.StatusBadRequest, "capacity cannot be less than current confirmed registrations")
				return
			}
			newCap = sql.NullInt64{Int64: int64(*req.Capacity), Valid: true}
			// Adjust slots_remaining proportionally.
			if cap.Valid {
				used := cap.Int64 - slots.Int64
				newSlots = sql.NullInt64{Int64: int64(*req.Capacity) - used, Valid: true}
				if newSlots.Int64 < 0 {
					newSlots.Int64 = 0
				}
			} else {
				newSlots = sql.NullInt64{Int64: int64(*req.Capacity), Valid: true}
			}
		}
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(r.Context(),
		`UPDATE events SET title=?, description=?, location=?, start_time=?, end_time=?,
		  capacity=?, slots_remaining=?, updated_at=?
		 WHERE id=?`,
		e.Title, e.Description, e.Location, e.StartTime, e.EndTime,
		newCap, newSlots, now, id,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not update event")
		return
	}

	// Replace skill links if provided.
	if req.SkillIDs != nil {
		if _, err = tx.ExecContext(r.Context(), `DELETE FROM event_skills WHERE event_id = ?`, id); err != nil {
			respondError(w, http.StatusInternalServerError, "could not update skills")
			return
		}
		for _, skillID := range *req.SkillIDs {
			if _, err = tx.ExecContext(r.Context(),
				`INSERT OR IGNORE INTO event_skills (event_id, skill_id) VALUES (?, ?)`, id, skillID,
			); err != nil {
				respondError(w, http.StatusInternalServerError, "could not link skill")
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	// Re-read the full event to return accurate capacity/slots values.
	var updated models.Event
	_ = s.DB.QueryRowContext(r.Context(),
		`SELECT id, host_id, title, description, location, start_time, end_time, status,
		        capacity, slots_remaining, created_at, updated_at
		 FROM events WHERE id = ?`, id,
	).Scan(&updated.ID, &updated.HostID, &updated.Title, &updated.Description,
		&updated.Location, &updated.StartTime, &updated.EndTime, &updated.Status,
		&updated.Capacity, &updated.SlotsRemaining, &updated.CreatedAt, &updated.UpdatedAt)
	updated.Skills = s.fetchEventSkills(r, id)
	respond(w, http.StatusOK, updated)
}

// UnregisterFromEvent handles DELETE /api/events/{id}/register  (student only)
//
// Removes the student's registration from the event and restores the slot if
// the registration was confirmed and the event has a capacity limit.
func (s *Server) UnregisterFromEvent(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	studentID := middleware.GetUserID(r.Context())

	// Find the registration.
	var regID string
	var regStatus models.RegistrationStatus
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT id, status FROM registrations WHERE event_id = ? AND student_id = ?`,
		eventID, studentID,
	).Scan(&regID, &regStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "registration not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(r.Context(),
		`DELETE FROM registrations WHERE id = ?`, regID,
	); err != nil {
		respondError(w, http.StatusInternalServerError, "could not remove registration")
		return
	}

	// Restore slot if this was a confirmed registration for a capacity-limited event.
	if regStatus == models.RegistrationConfirmed {
		tx.ExecContext(r.Context(), //nolint:errcheck
			`UPDATE events SET slots_remaining = slots_remaining + 1, updated_at = ?
			 WHERE id = ? AND capacity IS NOT NULL`,
			time.Now().UTC(), eventID,
		)
	}

	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// KickRegistration handles DELETE /api/events/{id}/registrations/{reg_id}  (host only)
//
// Allows the host to remove any registration (regardless of status) and
// restore the slot if applicable.
func (s *Server) KickRegistration(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("id")
	regID := r.PathValue("reg_id")
	hostID := middleware.GetUserID(r.Context())

	// Verify host ownership.
	var dbHostID string
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT host_id FROM events WHERE id = ?`, eventID,
	).Scan(&dbHostID)
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

	// Get the registration status before deleting.
	var regStatus models.RegistrationStatus
	err = s.DB.QueryRowContext(r.Context(),
		`SELECT status FROM registrations WHERE id = ? AND event_id = ?`, regID, eventID,
	).Scan(&regStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusNotFound, "registration not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(r.Context(),
		`DELETE FROM registrations WHERE id = ?`, regID,
	); err != nil {
		respondError(w, http.StatusInternalServerError, "could not remove registration")
		return
	}

	if regStatus == models.RegistrationConfirmed {
		tx.ExecContext(r.Context(), //nolint:errcheck
			`UPDATE events SET slots_remaining = slots_remaining + 1, updated_at = ?
			 WHERE id = ? AND capacity IS NOT NULL`,
			time.Now().UTC(), eventID,
		)
	}

	if err := tx.Commit(); err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// fetchEventSkills is an internal helper used by CreateEvent, ListEvents and
// GetEvent to load the skills attached to an event via the event_skills join table.
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
