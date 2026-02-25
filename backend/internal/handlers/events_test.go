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
