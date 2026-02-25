package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
)

// makeCheckInPayload builds a valid CheckInPayload JSON string for testing.
// It generates a signed check-in token using the test server's secret.
func makeCheckInPayload(t *testing.T, eventID, checkInCode, secret string) string {
	t.Helper()
	token, err := auth.GenerateCheckInToken(eventID, checkInCode, secret)
	if err != nil {
		t.Fatalf("makeCheckInPayload: %v", err)
	}
	return fmt.Sprintf(`{"token":%q}`, token)
}

func TestSyncAttendance_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	skillID := seedSkill(t, srv, "Python")
	eventID, checkInCode := seedEvent(t, srv, companyID)

	// Link skill to event
	_, err := srv.DB.Exec(`INSERT INTO event_skills (event_id, skill_id) VALUES (?, ?)`, eventID, skillID)
	if err != nil {
		t.Fatalf("link skill: %v", err)
	}

	payload := makeCheckInPayload(t, eventID, checkInCode, testSecret)
	localID := "local-001"

	req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{
			{LocalID: localID, EventID: eventID, Payload: payload},
		},
	}))
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.SyncAttendance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp models.SyncAttendanceResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Status != models.AttendanceVerified {
		t.Errorf("status: got %q, want verified. message: %s", resp.Results[0].Status, resp.Results[0].Message)
	}

	// Verify skill was awarded
	var count int
	srv.DB.QueryRow(`SELECT COUNT(*) FROM user_skills WHERE user_id = ? AND skill_id = ?`, studentID, skillID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 user_skill record, got %d", count)
	}
}

func TestSyncAttendance_InvalidSignature(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, checkInCode := seedEvent(t, srv, companyID)

	// Build a token signed with the WRONG secret — simulates a forged QR.
	badToken, _ := auth.GenerateCheckInToken(eventID, checkInCode, "wrong-secret")
	payload := fmt.Sprintf(`{"token":%q}`, badToken)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{
			{LocalID: "local-002", EventID: eventID, Payload: payload},
		},
	}))
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.SyncAttendance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp models.SyncAttendanceResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Results[0].Status != models.AttendanceRejected {
		t.Errorf("expected rejected, got %q", resp.Results[0].Status)
	}
}

func TestSyncAttendance_Idempotent(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, checkInCode := seedEvent(t, srv, companyID)

	payload := makeCheckInPayload(t, eventID, checkInCode, testSecret)
	syncReq := models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{
			{LocalID: "local-003", EventID: eventID, Payload: payload},
		},
	}

	// Sync twice — should be idempotent
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, syncReq))
		req = ctxWithUser(req, studentID, "student")
		rec := httptest.NewRecorder()
		srv.SyncAttendance(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d", i+1, rec.Code)
		}
		var resp models.SyncAttendanceResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Results[0].Status != models.AttendanceVerified {
			t.Errorf("attempt %d: expected verified, got %q", i+1, resp.Results[0].Status)
		}
	}
}

// TestSyncAttendance_TokenExpiredButSyncedLate verifies the core design:
// a token whose exp has passed is still accepted at sync time because the
// server only checks the JWT signature, not the expiry, during sync.
// The exp only matters at scan time (i.e. when the host's QR was live).
func TestSyncAttendance_TokenExpiredButSyncedLate(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, checkInCode := seedEvent(t, srv, companyID)

	// Manually build a token that expired 1 week ago — simulates a student
	// who scanned the QR during the live window but only connects now.
	// We sign it with the correct secret so the signature is valid.
	expiredToken, err := auth.GenerateCheckInTokenWithExpiry(
		eventID, checkInCode, testSecret,
		time.Now().Add(-7*24*time.Hour),                                // iat in the past
		time.Now().Add(-7*24*time.Hour).Add(auth.CheckInTokenDuration), // exp also in the past
	)
	if err != nil {
		t.Fatalf("GenerateCheckInTokenWithExpiry: %v", err)
	}

	payload := fmt.Sprintf(`{"token":%q}`, expiredToken)
	req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{
			{LocalID: "local-late", EventID: eventID, Payload: payload},
		},
	}))
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.SyncAttendance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp models.SyncAttendanceResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	// Must be ACCEPTED — signature is valid, sync is just late.
	if resp.Results[0].Status != models.AttendanceVerified {
		t.Errorf("expected verified for late sync, got %q: %s", resp.Results[0].Status, resp.Results[0].Message)
	}
}

// TestSyncAttendance_WrongSecret verifies that a token signed by a different
// server (or a tampered token) is rejected even if the exp hasn't passed.
func TestSyncAttendance_WrongSecret(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, checkInCode := seedEvent(t, srv, companyID)

	// Token issued by a "different server" — wrong signing secret.
	foreignToken, _ := auth.GenerateCheckInToken(eventID, checkInCode, "different-server-secret")
	payload := fmt.Sprintf(`{"token":%q}`, foreignToken)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{
			{LocalID: "local-foreign", EventID: eventID, Payload: payload},
		},
	}))
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.SyncAttendance(rec, req)

	var resp models.SyncAttendanceResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Results[0].Status != models.AttendanceRejected {
		t.Errorf("expected rejected for foreign-signed token, got %q", resp.Results[0].Status)
	}
}

func TestGetMySkills(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	skillID := seedSkill(t, srv, "Docker")
	eventID, checkInCode := seedEvent(t, srv, companyID)
	srv.DB.Exec(`INSERT INTO event_skills (event_id, skill_id) VALUES (?, ?)`, eventID, skillID)

	// Sync attendance to award skill
	payload := makeCheckInPayload(t, eventID, checkInCode, testSecret)
	syncReq := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{{LocalID: "local-005", EventID: eventID, Payload: payload}},
	}))
	syncReq = ctxWithUser(syncReq, studentID, "student")
	srv.SyncAttendance(httptest.NewRecorder(), syncReq)

	req := httptest.NewRequest(http.MethodGet, "/api/users/me/skills", nil)
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.GetMySkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var skills []models.UserSkill
	json.NewDecoder(rec.Body).Decode(&skills)
	if len(skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(skills))
	}
}

// TestSyncAutoRegisters verifies that scanning a QR while unregistered
// automatically creates a confirmed registration on sync.
func TestSyncAutoRegisters(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, checkInCode := seedEvent(t, srv, companyID)

	// Confirm no prior registration.
	var count int
	srv.DB.QueryRow(`SELECT COUNT(*) FROM registrations WHERE event_id=? AND student_id=?`, eventID, studentID).Scan(&count)
	if count != 0 {
		t.Fatal("expected no prior registration")
	}

	payload := makeCheckInPayload(t, eventID, checkInCode, testSecret)
	req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{{LocalID: "local-auto", EventID: eventID, Payload: payload}},
	}))
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.SyncAttendance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	srv.DB.QueryRow(`SELECT COUNT(*) FROM registrations WHERE event_id=? AND student_id=?`, eventID, studentID).Scan(&count)
	if count != 1 {
		t.Errorf("expected auto-registration to be created, got count=%d", count)
	}
	var regStatus string
	srv.DB.QueryRow(`SELECT status FROM registrations WHERE event_id=? AND student_id=?`, eventID, studentID).Scan(&regStatus)
	if regStatus != "confirmed" {
		t.Errorf("expected confirmed, got %q", regStatus)
	}
}

// TestSyncAutoRegisters_ConflictWhenFull verifies that scanning a QR for a
// fully-booked event creates a conflict_pending registration.
func TestSyncAutoRegisters_ConflictWhenFull(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	student1 := seedStudentUser(t, srv)
	student2 := seedStudentUser(t, srv)

	// Capacity 1, already used by student1.
	eventID, checkInCode := seedEventWithCapacity(t, srv, companyID, 1)

	// Student1 registers online first, consuming the only slot.
	req1 := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/register", nil)
	req1.SetPathValue("id", eventID)
	req1 = ctxWithUser(req1, student1, "student")
	srv.RegisterForEvent(httptest.NewRecorder(), req1)

	// Student2 syncs an offline QR scan.
	payload := makeCheckInPayload(t, eventID, checkInCode, testSecret)
	req2 := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{{LocalID: "local-conflict", EventID: eventID, Payload: payload}},
	}))
	req2 = ctxWithUser(req2, student2, "student")
	rec2 := httptest.NewRecorder()
	srv.SyncAttendance(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var regStatus string
	srv.DB.QueryRow(`SELECT status FROM registrations WHERE event_id=? AND student_id=?`, eventID, student2).Scan(&regStatus)
	if regStatus != "conflict_pending" {
		t.Errorf("expected conflict_pending, got %q", regStatus)
	}
}
