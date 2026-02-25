package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
)

const testSecret = "middleware-test-secret"

func TestCORS_SetsHeaders(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("ACAO header: got %q, want *", got)
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status: got %d, want 204", rec.Code)
	}
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	handler := Authenticate(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	token, err := auth.GenerateToken("user-1", "student", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	var capturedID string
	handler := Authenticate(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if capturedID != "user-1" {
		t.Errorf("user_id: got %q, want user-1", capturedID)
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	token, _ := auth.GenerateToken("user-2", "student", testSecret)

	handler := Authenticate(testSecret)(
		RequireRole("company")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}
