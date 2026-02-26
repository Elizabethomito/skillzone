package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/models"
	"github.com/google/uuid"
)

// seedCompanyUser inserts a company user and returns their ID.
func seedCompanyUser(t *testing.T, srv *Server) string {
	t.Helper()
	id := uuid.NewString()
	_, err := srv.DB.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, 'company')`,
		id, fmt.Sprintf("company-%s@test.com", id), "hash", "Test Company",
	)
	if err != nil {
		t.Fatalf("seedCompanyUser: %v", err)
	}
	return id
}

// seedStudentUser inserts a student user and returns their ID.
func seedStudentUser(t *testing.T, srv *Server) string {
	t.Helper()
	id := uuid.NewString()
	_, err := srv.DB.Exec(
		`INSERT INTO users (id, email, password_hash, name, role) VALUES (?, ?, ?, ?, 'student')`,
		id, fmt.Sprintf("student-%s@test.com", id), "hash", "Test Student",
	)
	if err != nil {
		t.Fatalf("seedStudentUser: %v", err)
	}
	return id
}

// seedSkill inserts a skill and returns its ID.
func seedSkill(t *testing.T, srv *Server, name string) string {
	t.Helper()
	id := uuid.NewString()
	_, err := srv.DB.Exec(
		`INSERT INTO skills (id, name, description) VALUES (?, ?, ?)`,
		id, name, "A skill",
	)
	if err != nil {
		t.Fatalf("seedSkill: %v", err)
	}
	return id
}

// seedEvent inserts an event and returns its ID and check_in_code.
func seedEvent(t *testing.T, srv *Server, hostID string) (eventID, checkInCode string) {
	t.Helper()
	eventID = uuid.NewString()
	checkInCode = uuid.NewString()
	_, err := srv.DB.Exec(
		`INSERT INTO events (id, host_id, title, description, location, start_time, end_time, status, check_in_code)
		 VALUES (?, ?, 'Test Event', '', '', ?, ?, 'active', ?)`,
		eventID, hostID,
		time.Now().UTC(),
		time.Now().Add(2*time.Hour).UTC(),
		checkInCode,
	)
	if err != nil {
		t.Fatalf("seedEvent: %v", err)
	}
	return
}

func TestCreateEvent_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	skillID := seedSkill(t, srv, "Go Programming")

	req := httptest.NewRequest(http.MethodPost, "/api/events", jsonBody(t, models.CreateEventRequest{
		Title:       "Go Workshop",
		Description: "Learn Go",
		Location:    "Nairobi",
		StartTime:   time.Now().Add(1 * time.Hour),
		EndTime:     time.Now().Add(3 * time.Hour),
		SkillIDs:    []string{skillID},
	}))
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.CreateEvent(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var e models.Event
	if err := json.NewDecoder(rec.Body).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e.Title != "Go Workshop" {
		t.Errorf("title: got %q", e.Title)
	}
	if len(e.Skills) != 1 {
		t.Errorf("skills: expected 1, got %d", len(e.Skills))
	}
}

func TestCreateEvent_MissingTitle(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/api/events", jsonBody(t, models.CreateEventRequest{
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(3 * time.Hour),
	}))
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.CreateEvent(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestListEvents(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	seedEvent(t, srv, companyID)
	seedEvent(t, srv, companyID)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()
	srv.ListEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var events []models.Event
	json.NewDecoder(rec.Body).Decode(&events)
	if len(events) < 2 {
		t.Errorf("expected at least 2 events, got %d", len(events))
	}
}

func TestRegisterForEvent_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/register", nil)
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.RegisterForEvent(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterForEvent_Idempotent(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/register", nil)
		req.SetPathValue("id", eventID)
		req = ctxWithUser(req, studentID, "student")
		rec := httptest.NewRecorder()
		srv.RegisterForEvent(rec, req)
		// Both calls should succeed (INSERT OR IGNORE)
		if rec.Code != http.StatusCreated {
			t.Errorf("attempt %d: expected 201, got %d", i+1, rec.Code)
		}
	}
}

// seedEventWithCapacity inserts an event with a capacity limit for slot tests.
func seedEventWithCapacity(t *testing.T, srv *Server, hostID string, capacity int) (eventID, checkInCode string) {
	t.Helper()
	eventID = uuid.NewString()
	checkInCode = uuid.NewString()
	_, err := srv.DB.Exec(
		`INSERT INTO events (id, host_id, title, description, location, start_time, end_time, status, check_in_code, capacity, slots_remaining)
		 VALUES (?, ?, 'Internship', '', '', ?, ?, 'upcoming', ?, ?, ?)`,
		eventID, hostID,
		time.Now().Add(24*time.Hour).UTC(),
		time.Now().Add(48*time.Hour).UTC(),
		checkInCode, capacity, capacity,
	)
	if err != nil {
		t.Fatalf("seedEventWithCapacity: %v", err)
	}
	return
}

func TestRegisterForEvent_CapacityConflict(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	student1 := seedStudentUser(t, srv)
	student2 := seedStudentUser(t, srv)

	// Event with only 1 slot.
	eventID, _ := seedEventWithCapacity(t, srv, companyID, 1)

	register := func(studentID string) models.Registration {
		req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/register", nil)
		req.SetPathValue("id", eventID)
		req = ctxWithUser(req, studentID, "student")
		rec := httptest.NewRecorder()
		srv.RegisterForEvent(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var reg models.Registration
		json.NewDecoder(rec.Body).Decode(&reg)
		return reg
	}

	reg1 := register(student1)
	reg2 := register(student2)

	if reg1.Status != models.RegistrationConfirmed {
		t.Errorf("first registration should be confirmed, got %q", reg1.Status)
	}
	if reg2.Status != models.RegistrationConflictPending {
		t.Errorf("second registration should be conflict_pending, got %q", reg2.Status)
	}
}

func TestUpdateEventStatus(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	req := httptest.NewRequest(http.MethodPatch, "/api/events/"+eventID+"/status",
		jsonBody(t, models.UpdateEventStatusRequest{Status: models.EventStatusCompleted}))
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.UpdateEventStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Verify DB was updated.
	var status string
	srv.DB.QueryRow(`SELECT status FROM events WHERE id = ?`, eventID).Scan(&status)
	if status != "completed" {
		t.Errorf("expected status=completed, got %q", status)
	}
}

func TestResolveRegistrationConflict(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEventWithCapacity(t, srv, companyID, 0) // 0 slots → conflict immediately

	// Force a conflict_pending registration directly.
	regID := uuid.NewString()
	_, err := srv.DB.Exec(
		`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'conflict_pending')`,
		regID, eventID, studentID,
	)
	if err != nil {
		t.Fatalf("insert conflict reg: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/events/"+eventID+"/registrations/"+regID,
		jsonBody(t, models.ResolveConflictRequest{Action: "confirm"}))
	req.SetPathValue("id", eventID)
	req.SetPathValue("reg_id", regID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.ResolveRegistrationConflict(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var newStatus string
	srv.DB.QueryRow(`SELECT status FROM registrations WHERE id = ?`, regID).Scan(&newStatus)
	if newStatus != "confirmed" {
		t.Errorf("expected confirmed, got %q", newStatus)
	}
}

func TestGetEventRegistrations(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	srv.DB.Exec(`INSERT OR IGNORE INTO registrations (id, event_id, student_id) VALUES (?, ?, ?)`,
		uuid.NewString(), eventID, studentID)

	req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID+"/registrations", nil)
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.GetEventRegistrations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("expected 1 registration, got %d", len(result))
	}
}

func TestUpdateEvent_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	newTitle := "Updated Title"
	newDesc := "Updated description"
	req := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID,
		jsonBody(t, models.UpdateEventRequest{
			Title:       newTitle,
			Description: &newDesc,
		}))
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.UpdateEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var e models.Event
	json.NewDecoder(rec.Body).Decode(&e)
	if e.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, e.Title)
	}
	if e.Description != newDesc {
		t.Errorf("expected description %q, got %q", newDesc, e.Description)
	}
}

func TestUpdateEvent_ForbiddenForNonHost(t *testing.T) {
	srv := newTestServer(t)
	hostID := seedCompanyUser(t, srv)
	otherCompanyID := seedCompanyUser(t, srv)
	eventID, _ := seedEvent(t, srv, hostID)

	req := httptest.NewRequest(http.MethodPut, "/api/events/"+eventID,
		jsonBody(t, models.UpdateEventRequest{Title: "Hijacked"}))
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, otherCompanyID, "company")
	rec := httptest.NewRecorder()
	srv.UpdateEvent(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestUnregisterFromEvent_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	// Register first.
	regID := uuid.NewString()
	srv.DB.Exec(`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'confirmed')`,
		regID, eventID, studentID)

	req := httptest.NewRequest(http.MethodDelete, "/api/events/"+eventID+"/register", nil)
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.UnregisterFromEvent(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	var count int
	srv.DB.QueryRow(`SELECT COUNT(*) FROM registrations WHERE id = ?`, regID).Scan(&count)
	if count != 0 {
		t.Errorf("expected registration to be deleted, but still exists")
	}
}

func TestUnregisterFromEvent_RestoresSlot(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEventWithCapacity(t, srv, companyID, 1)

	// Consume the slot.
	srv.DB.Exec(`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'confirmed')`,
		uuid.NewString(), eventID, studentID)
	srv.DB.Exec(`UPDATE events SET slots_remaining = 0 WHERE id = ?`, eventID)

	req := httptest.NewRequest(http.MethodDelete, "/api/events/"+eventID+"/register", nil)
	req.SetPathValue("id", eventID)
	req = ctxWithUser(req, studentID, "student")
	rec := httptest.NewRecorder()
	srv.UnregisterFromEvent(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	var slots int
	srv.DB.QueryRow(`SELECT slots_remaining FROM events WHERE id = ?`, eventID).Scan(&slots)
	if slots != 1 {
		t.Errorf("expected slots_remaining=1 after unregister, got %d", slots)
	}
}

func TestKickRegistration_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	regID := uuid.NewString()
	srv.DB.Exec(`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'confirmed')`,
		regID, eventID, studentID)

	req := httptest.NewRequest(http.MethodDelete, "/api/events/"+eventID+"/registrations/"+regID, nil)
	req.SetPathValue("id", eventID)
	req.SetPathValue("reg_id", regID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.KickRegistration(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Now the registration should be marked as rejected (not deleted).
	var status string
	err := srv.DB.QueryRow(`SELECT status FROM registrations WHERE id = ?`, regID).Scan(&status)
	if err != nil {
		t.Fatalf("expected registration to still exist: %v", err)
	}
	if status != "rejected" {
		t.Errorf("expected status=rejected, got %q", status)
	}
}

func TestKickRegistration_AlreadyRejected(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	regID := uuid.NewString()
	srv.DB.Exec(`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'rejected')`,
		regID, eventID, studentID)

	req := httptest.NewRequest(http.MethodDelete, "/api/events/"+eventID+"/registrations/"+regID, nil)
	req.SetPathValue("id", eventID)
	req.SetPathValue("reg_id", regID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.KickRegistration(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 when already rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestReAddRegistration_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	regID := uuid.NewString()
	srv.DB.Exec(`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'rejected')`,
		regID, eventID, studentID)

	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/registrations/"+regID+"/readd", nil)
	req.SetPathValue("id", eventID)
	req.SetPathValue("reg_id", regID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.ReAddRegistration(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var status string
	srv.DB.QueryRow(`SELECT status FROM registrations WHERE id = ?`, regID).Scan(&status)
	if status != "confirmed" {
		t.Errorf("expected status=confirmed after re-add, got %q", status)
	}
}

func TestReAddRegistration_NotRejected(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	eventID, _ := seedEvent(t, srv, companyID)

	regID := uuid.NewString()
	srv.DB.Exec(`INSERT INTO registrations (id, event_id, student_id, status) VALUES (?, ?, ?, 'confirmed')`,
		regID, eventID, studentID)

	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/registrations/"+regID+"/readd", nil)
	req.SetPathValue("id", eventID)
	req.SetPathValue("reg_id", regID)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.ReAddRegistration(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 when not rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSearchStudents_NoFilter(t *testing.T) {
	srv := newTestServer(t)
	_ = seedStudentUser(t, srv)
	_ = seedStudentUser(t, srv)
	companyID := seedCompanyUser(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/users/students", nil)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) < 2 {
		t.Errorf("expected at least 2 students, got %d", len(result))
	}
}

func TestSearchStudents_BySkill(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentWithSkill := seedStudentUser(t, srv)
	_ = seedStudentUser(t, srv) // student without the skill
	skillID := seedSkill(t, srv, "TestSkill-"+uuid.NewString())
	eventID, _ := seedEvent(t, srv, companyID)

	// Award the skill to the first student.
	srv.DB.Exec(`INSERT OR IGNORE INTO event_skills (event_id, skill_id) VALUES (?, ?)`, eventID, skillID)
	srv.DB.Exec(`INSERT INTO user_skills (id, user_id, skill_id, event_id, awarded_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.NewString(), studentWithSkill, skillID, eventID, time.Now().UTC())

	req := httptest.NewRequest(http.MethodGet, "/api/users/students?skill_id="+skillID, nil)
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) != 1 {
		t.Errorf("expected exactly 1 student with that skill, got %d", len(result))
	}
}
