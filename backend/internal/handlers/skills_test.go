package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Elizabethomito/skillzone/backend/internal/models"
)

func TestCreateSkill_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/api/skills", jsonBody(t, map[string]string{
		"name":        "Kubernetes",
		"description": "Container orchestration",
	}))
	req = ctxWithUser(req, companyID, "company")
	rec := httptest.NewRecorder()
	srv.CreateSkill(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var sk models.Skill
	json.NewDecoder(rec.Body).Decode(&sk)
	if sk.Name != "Kubernetes" {
		t.Errorf("name: got %q", sk.Name)
	}
}

func TestCreateSkill_Duplicate(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/skills", jsonBody(t, map[string]string{
			"name": "React",
		}))
		req = ctxWithUser(req, companyID, "company")
		rec := httptest.NewRecorder()
		srv.CreateSkill(rec, req)
		if i == 1 && rec.Code != http.StatusConflict {
			t.Errorf("expected 409 on duplicate, got %d", rec.Code)
		}
	}
}

func TestListSkills(t *testing.T) {
	srv := newTestServer(t)
	seedSkill(t, srv, "TypeScript")
	seedSkill(t, srv, "Rust")

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.ListSkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var skills []models.Skill
	json.NewDecoder(rec.Body).Decode(&skills)
	if len(skills) < 2 {
		t.Errorf("expected at least 2 skills, got %d", len(skills))
	}
}

func TestGetUserPublicProfile_Success(t *testing.T) {
	srv := newTestServer(t)
	companyID := seedCompanyUser(t, srv)
	studentID := seedStudentUser(t, srv)
	skillID := seedSkill(t, srv, "PublicSkill")
	eventID, _ := seedEvent(t, srv, companyID)

	// Link skill to event and award to student.
	srv.DB.Exec(`INSERT OR IGNORE INTO event_skills (event_id, skill_id) VALUES (?, ?)`, eventID, skillID)
	srv.DB.Exec(`INSERT INTO user_skills (id, user_id, skill_id, event_id, awarded_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		"badge-1", studentID, skillID, eventID)

	req := httptest.NewRequest(http.MethodGet, "/api/users/"+studentID+"/profile", nil)
	req.SetPathValue("id", studentID)
	rec := httptest.NewRecorder()
	srv.GetUserPublicProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var profile map[string]any
	json.NewDecoder(rec.Body).Decode(&profile)
	if profile["id"] != studentID {
		t.Errorf("expected profile id=%s, got %v", studentID, profile["id"])
	}
	badges, ok := profile["badges"].([]any)
	if !ok || len(badges) == 0 {
		t.Errorf("expected at least 1 badge in profile, got %v", profile["badges"])
	}
}

func TestGetUserPublicProfile_NotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/users/nonexistent/profile", nil)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()
	srv.GetUserPublicProfile(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}
