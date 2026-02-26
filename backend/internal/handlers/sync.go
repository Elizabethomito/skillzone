package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
) // SyncAttendance handles POST /api/sync/attendance  (student only)
// This is the heart of the "local-first" design. Here is the full flow:
//
//  1. HOST SIDE (online, at the event):
//     Company calls GET /api/events/{id}/checkin-code to get the check_in_code.
//     Their PWA builds a CheckInPayload JSON:
//     { "event_id": "...", "host_sig": "<check_in_code>", "timestamp": <unix> }
//     and displays it as a QR code on a screen.
//
//  2. STUDENT SIDE (offline is fine):
//     Student scans the QR with their PWA. The PWA stores the raw payload JSON
//     in IndexedDB (via Dexie.js) as ATTENDANCE_PENDING.
//
//  3. SYNC (when student is back online):
//     Student's PWA sends all PENDING records to this endpoint in one batch.
//     For each record the server verifies the payload and awards badges.
//     The server echoes back each record's LocalID so the PWA can update
//     its IndexedDB records to VERIFIED or REJECTED.
//
// LEARNING NOTE — batch processing
// We process every record even if some fail; the response reports individual
// statuses. This means a student who has 3 pending check-ins doesn't lose
// 2 of them because the 1st one had a bad payload.
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

// processAttendanceRecord validates and persists a single offline check-in.
//
// It is deliberately separated from SyncAttendance so it can be unit-tested
// directly (see sync_test.go) and so the loop in SyncAttendance stays clean.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — the new token-based verification model
// ────────────────────────────────────────────────────────────────────
// v1 (old): the QR code contained a raw check_in_code UUID + unix timestamp.
//
//	The server rejected syncs where time.Since(timestamp) > 24 h, which
//	forced students with poor connectivity to sync quickly.
//
// v2 (new): the QR code contains a signed JWT (CheckInPayload.Token).
//
//	The JWT has a 6-hour exp, enforcing the SCAN window — you must have
//	scanned while the QR was live on screen.  But the JWT never expires
//	for the SERVER: once a student holds a valid signed token we accept
//	their sync at any point in the future.  There is no wall-clock check
//	at sync time.
//
// Why is this safe?
//   - The JWT is signed with the server secret — it cannot be forged.
//   - The JWT's exp was set when the host generated the QR.  A student
//     scanning after exp would get a JWT that fails ParseCheckInToken.
//   - A student who scanned during the live window holds a token whose
//     signature is permanently valid (we verify sig only, not exp at sync).
//   - We achieve this by calling jwt.ParseWithClaims with a custom
//     parser option (jwt.WithoutClaimsValidation) inside ParseCheckInToken
//     — see auth/jwt.go.
func (s *Server) processAttendanceRecord(r *http.Request, studentID string, rec models.AttendanceSyncRecord) models.SyncResult {
	// Helper to build a rejection result in one line.
	fail := func(msg string) models.SyncResult {
		return models.SyncResult{LocalID: rec.LocalID, Status: models.AttendanceRejected, Message: msg}
	}

	// Step 1 — Parse the QR payload the student's PWA captured.
	var payload models.CheckInPayload
	if err := json.Unmarshal([]byte(rec.Payload), &payload); err != nil {
		return fail("invalid payload JSON")
	}

	if payload.Token == "" {
		return fail("payload missing token")
	}

	// Step 2 — Verify the signed check-in token.
	// ParseCheckInToken checks the JWT signature (was this signed by our server?)
	// but NOT the expiry — the exp only controls the scan window, which already
	// closed when the student's device stored the payload.
	claims, err := auth.ParseCheckInToken(payload.Token, s.Secret)
	if err != nil {
		return fail("invalid check-in token: " + err.Error())
	}

	// Step 3 — The token's event_id must match the outer record's event_id.
	// This guards against a student copy-pasting the wrong QR payload.
	if claims.EventID != rec.EventID {
		return fail("token event_id does not match record event_id")
	}

	// Step 4 — Confirm the event still exists in the database.
	// (The host_sig inside the token is already verified by the JWT signature,
	// so we don't need a separate database lookup to confirm it.)
	var exists bool
	err = s.DB.QueryRowContext(r.Context(),
		`SELECT 1 FROM events WHERE id = ?`, rec.EventID,
	).Scan(&exists)
	if err != nil || !exists {
		return fail("event not found")
	}

	// Step 5 — Upsert the attendance record (idempotent).
	// ON CONFLICT ... DO UPDATE means a retry on bad network just refreshes
	// the updated_at timestamp without creating a duplicate row.
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

	// Step 6 — Auto-register the student if they haven't already.
	// This handles the scenario where a student scans the QR without
	// having pre-registered online.  Same slot-aware logic as RegisterForEvent.
	if err := s.upsertRegistration(r, studentID, rec.EventID, now); err != nil {
		return fail("could not record registration: " + err.Error())
	}

	// Step 7 — Award badges (also idempotent via INSERT OR IGNORE).
	if err := s.awardSkills(r, studentID, rec.EventID); err != nil {
		return fail("could not award skills: " + err.Error())
	}

	return models.SyncResult{
		LocalID: rec.LocalID,
		Status:  models.AttendanceVerified,
		Message: "attendance verified and skills awarded",
	}
}

// upsertRegistration ensures a registration row exists for the student at the event.
// Called from processAttendanceRecord so a QR scan auto-registers the student
// even if they never pressed "Register" while online.
//
// Slot logic (mirrors RegisterForEvent):
//   - If capacity is NULL or slots remain → confirmed.
//   - If no slots remain → conflict_pending (host dashboard will show this).
//   - If already registered → no change (INSERT OR IGNORE).
func (s *Server) upsertRegistration(r *http.Request, studentID, eventID string, now time.Time) error {
	// Read capacity state.
	var capacity, slotsRemaining sql.NullInt64
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT capacity, slots_remaining FROM events WHERE id = ?`, eventID,
	).Scan(&capacity, &slotsRemaining)
	if err != nil {
		return err
	}

	regStatus := "confirmed"
	if capacity.Valid && capacity.Int64 > 0 && slotsRemaining.Valid && slotsRemaining.Int64 <= 0 {
		regStatus = "conflict_pending"
	}

	result, err := s.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
		 VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), eventID, studentID, now, regStatus,
	)
	if err != nil {
		return err
	}

	// Only decrement slots when a brand-new confirmed registration was inserted.
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 && regStatus == "confirmed" && capacity.Valid {
		_, err = s.DB.ExecContext(r.Context(),
			`UPDATE events SET slots_remaining = slots_remaining - 1, updated_at = ?
			 WHERE id = ? AND slots_remaining > 0`,
			now, eventID,
		)
		return err
	}
	return nil
}

// awardSkills inserts a UserSkill row for each skill linked to the event.
//
// INSERT OR IGNORE means: if the student already has the badge (they synced
// before), skip silently. The UNIQUE constraint is on (user_id, skill_id, event_id).
// This makes awardSkills safe to call multiple times for the same student+event.
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

// GetMySkills handles GET /api/users/me/skills  (student only)
//
// Uses a JOIN to fetch skill details in one query instead of N queries.
// The result is sorted newest-first so the PWA can display recently earned
// badges at the top.
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
		us.Skill = &models.Skill{} // allocate so Scan can fill the nested pointer
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

	// Return [] not null for an empty badge collection.
	if userSkills == nil {
		userSkills = []models.UserSkill{}
	}
	respond(w, http.StatusOK, userSkills)
}

// GetMyRegistrations handles GET /api/users/me/registrations  (student only)
//
// Returns the student's registered events with event details embedded,
// avoiding a second round-trip from the client. The anonymous struct
// RegWithEvent is defined inline because it's only used here.
func (s *Server) GetMyRegistrations(w http.ResponseWriter, r *http.Request) {
	studentID := middleware.GetUserID(r.Context())

	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT r.id, r.event_id, r.student_id, r.registered_at, r.status,
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

	// Inline type — embeds Registration so we inherit its JSON field names,
	// then adds the extra event fields from the JOIN.
	type RegWithEvent struct {
		models.Registration
		EventTitle  string             `json:"event_title"`
		StartTime   time.Time          `json:"start_time"`
		EndTime     time.Time          `json:"end_time"`
		EventStatus models.EventStatus `json:"event_status"`
		Location    string             `json:"location"`
	}

	var regs []RegWithEvent
	for rows.Next() {
		var reg RegWithEvent
		if err := rows.Scan(
			&reg.ID, &reg.EventID, &reg.StudentID, &reg.RegisteredAt, &reg.Status,
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
