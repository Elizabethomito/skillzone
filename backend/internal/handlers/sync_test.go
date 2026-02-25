package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/models"
)

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

	payload := fmt.Sprintf(
		`{"event_id":%q,"host_sig":%q,"timestamp":%d}`,
		eventID, checkInCode, time.Now().Unix(),
	)
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
	eventID, _ := seedEvent(t, srv, companyID)

	payload := fmt.Sprintf(
		`{"event_id":%q,"host_sig":"wrong-code","timestamp":%d}`,
		eventID, time.Now().Unix(),
	)

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

	payload := fmt.Sprintf(
		`{"event_id":%q,"host_sig":%q,"timestamp":%d}`,
		eventID, checkInCode, time.Now().Unix(),
	)
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

func TestSyncAttendance_ExpiredPayload(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, checkInCode := seedEvent(t, srv, companyID)

	// Payload timestamp is 25 hours ago
	staleTime := time.Now().Add(-25 * time.Hour).Unix()
	payload := fmt.Sprintf(
		`{"event_id":%q,"host_sig":%q,"timestamp":%d}`,
		eventID, checkInCode, staleTime,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/sync/attendance", jsonBody(t, models.SyncAttendanceRequest{
		Records: []models.AttendanceSyncRecord{
			{LocalID: "local-004", EventID: eventID, Payload: payload},
		},
	}))
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.SyncAttendance(rec, req)

	var resp models.SyncAttendanceResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Results[0].Status != models.AttendanceRejected {
		t.Errorf("expected rejected for stale payload, got %q", resp.Results[0].Status)
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
	payload := fmt.Sprintf(`{"event_id":%q,"host_sig":%q,"timestamp":%d}`, eventID, checkInCode, time.Now().Unix())
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
