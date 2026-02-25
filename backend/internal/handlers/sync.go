package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
)

// SyncAttendance handles POST /api/sync/attendance
// This is the core local-first sync endpoint. Students submit the QR payload
// they captured offline; the server verifies it against the event's check_in_code
// and awards skill badges.
func (s *Server) SyncAttendance(w http.ResponseWriter, r *http.Request) {
	studentID := middleware.GetUserID(r.Context())

	var req models.SyncAttendanceRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if len(req.Records) == 0 {
		respondError(w, http.StatusBadRequest, "no records to sync")
		return
	}

	results := make([]models.SyncResult, 0, len(req.Records))

	for _, rec := range req.Records {
		result := s.processAttendanceRecord(r, studentID, rec)
		results = append(results, result)
	}

	respond(w, http.StatusOK, models.SyncAttendanceResponse{Results: results})
}

// processAttendanceRecord handles a single offline check-in record.
func (s *Server) processAttendanceRecord(r *http.Request, studentID string, rec models.AttendanceSyncRecord) models.SyncResult {
	fail := func(msg string) models.SyncResult {
		return models.SyncResult{LocalID: rec.LocalID, Status: models.AttendanceRejected, Message: msg}
	}

	// Decode the QR payload the host device showed
	var payload models.CheckInPayload
	if err := json.Unmarshal([]byte(rec.Payload), &payload); err != nil {
		return fail("invalid payload JSON")
	}

	if payload.EventID == "" || payload.HostSig == "" {
		return fail("payload missing event_id or host_sig")
	}

	// Payload's event_id must match the record's event_id
	if payload.EventID != rec.EventID {
		return fail("payload event_id mismatch")
	}

	// Load the event and its check_in_code from the database
	var dbCheckInCode, hostID string
	var status models.EventStatus
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT check_in_code, host_id, status FROM events WHERE id = ?`, rec.EventID,
	).Scan(&dbCheckInCode, &hostID, &status)
	if err != nil {
		return fail("event not found")
	}

	// Verify the host_sig matches our stored check_in_code (the shared secret)
	if payload.HostSig != dbCheckInCode {
		return fail("invalid check-in signature")
	}

	// Reject stale payloads older than 24 hours
	payloadTime := time.Unix(payload.Timestamp, 0)
	if time.Since(payloadTime) > 24*time.Hour {
		return fail("check-in payload has expired")
	}

	// Upsert the attendance record (idempotent – safe to retry)
	attendanceID := uuid.NewString()
	now := time.Now().UTC()
	_, err = s.DB.ExecContext(r.Context(),
		`INSERT INTO attendances (id, event_id, student_id, payload, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'verified', ?, ?)
		 ON CONFLICT(event_id, student_id) DO UPDATE SET
		   status     = 'verified',
		   updated_at = excluded.updated_at`,
		attendanceID, rec.EventID, studentID, rec.Payload, now, now,
	)
	if err != nil {
		return fail("database error recording attendance")
	}

	// Award skill badges for this event (idempotent)
	if err := s.awardSkills(r, studentID, rec.EventID); err != nil {
		return fail("could not award skills: " + err.Error())
	}

	return models.SyncResult{
		LocalID: rec.LocalID,
		Status:  models.AttendanceVerified,
		Message: "attendance verified and skills awarded",
	}
}

// awardSkills inserts UserSkill rows for each skill linked to the event.
func (s *Server) awardSkills(r *http.Request, studentID, eventID string) error {
	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT skill_id FROM event_skills WHERE event_id = ?`, eventID)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().UTC()
	for rows.Next() {
		var skillID string
		if err := rows.Scan(&skillID); err != nil {
			return err
		}
		_, err = s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO user_skills (id, user_id, skill_id, event_id, awarded_at)
			 VALUES (?, ?, ?, ?, ?)`,
			uuid.NewString(), studentID, skillID, eventID, now,
		)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

// GetMySkills handles GET /api/users/me/skills  (student)
func (s *Server) GetMySkills(w http.ResponseWriter, r *http.Request) {
	studentID := middleware.GetUserID(r.Context())

	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT us.id, us.user_id, us.skill_id, us.event_id, us.awarded_at,
		        sk.name, sk.description
		 FROM user_skills us
		 JOIN skills sk ON sk.id = us.skill_id
		 WHERE us.user_id = ?
		 ORDER BY us.awarded_at DESC`, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	var userSkills []models.UserSkill
	for rows.Next() {
		var us models.UserSkill
		us.Skill = &models.Skill{}
		if err := rows.Scan(&us.ID, &us.UserID, &us.SkillID, &us.EventID, &us.AwardedAt,
			&us.Skill.Name, &us.Skill.Description); err != nil {
			respondError(w, http.StatusInternalServerError, "scan error")
			return
		}
		us.Skill.ID = us.SkillID
		userSkills = append(userSkills, us)
	}
	if err := rows.Err(); err != nil {
		respondError(w, http.StatusInternalServerError, "rows error")
		return
	}

	if userSkills == nil {
		userSkills = []models.UserSkill{}
	}
	respond(w, http.StatusOK, userSkills)
}

// GetMyRegistrations handles GET /api/users/me/registrations  (student)
func (s *Server) GetMyRegistrations(w http.ResponseWriter, r *http.Request) {
	studentID := middleware.GetUserID(r.Context())

	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT r.id, r.event_id, r.student_id, r.registered_at,
		        e.title, e.start_time, e.end_time, e.status, e.location
		 FROM registrations r
		 JOIN events e ON e.id = r.event_id
		 WHERE r.student_id = ?
		 ORDER BY e.start_time ASC`, studentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	type RegWithEvent struct {
		models.Registration
		EventTitle  string      `json:"event_title"`
		StartTime   time.Time   `json:"start_time"`
		EndTime     time.Time   `json:"end_time"`
		EventStatus models.EventStatus `json:"event_status"`
		Location    string      `json:"location"`
	}

	var regs []RegWithEvent
	for rows.Next() {
		var reg RegWithEvent
		if err := rows.Scan(
			&reg.ID, &reg.EventID, &reg.StudentID, &reg.RegisteredAt,
			&reg.EventTitle, &reg.StartTime, &reg.EndTime, &reg.EventStatus, &reg.Location,
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
		regs = []RegWithEvent{}
	}
	respond(w, http.StatusOK, regs)
}
