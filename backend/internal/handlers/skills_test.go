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
