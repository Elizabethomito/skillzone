package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Elizabethomito/skillzone/backend/internal/db"
	"github.com/Elizabethomito/skillzone/backend/internal/middleware"
	"github.com/Elizabethomito/skillzone/backend/internal/models"
)

const testSecret = "handler-test-secret"

var testDBCounter uint64

// newTestServer creates a Server backed by a unique in-memory SQLite database.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	// Each test gets its own named shared-cache memory DB so connections
	// in the pool all see the same tables without interfering across tests.
	id := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared&_foreign_keys=on", id)
	testDB, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("newTestServer: open db: %v", err)
	}
	t.Cleanup(func() { testDB.Close() })
	return &Server{DB: testDB, Secret: testSecret}
}

// jsonBody encodes v to JSON and returns a bytes.Buffer.
func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		t.Fatalf("jsonBody: %v", err)
	}
	return buf
}

// ctxWithUser attaches a user_id and role to a request's context (simulates Authenticate middleware).
func ctxWithUser(r *http.Request, userID, role string) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.ContextUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextRole, role)
	return r.WithContext(ctx)
}

// ---- Auth handler tests ----

func TestRegister_Success(t *testing.T) {
	srv := newTestServer(t)
	body := jsonBody(t, models.RegisterRequest{
		Email:    "alice@example.com",
		Password: "password123",
		Name:     "Alice",
		Role:     models.RoleStudent,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	rec := httptest.NewRecorder()
	srv.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp models.LoginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
	if resp.User.Email != "alice@example.com" {
		t.Errorf("email: got %q", resp.User.Email)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	srv := newTestServer(t)
	payload := models.RegisterRequest{
		Email:    "bob@example.com",
		Password: "password123",
		Name:     "Bob",
		Role:     models.RoleStudent,
	}
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", jsonBody(t, payload))
		rec := httptest.NewRecorder()
		srv.Register(rec, req)
		if i == 1 && rec.Code != http.StatusConflict {
			t.Errorf("expected 409, got %d", rec.Code)
		}
	}
}

func TestRegister_InvalidRole(t *testing.T) {
	srv := newTestServer(t)
	body := jsonBody(t, map[string]string{
		"email": "x@x.com", "password": "password123", "name": "X", "role": "admin",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	rec := httptest.NewRecorder()
	srv.Register(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	srv := newTestServer(t)
	// Register first
	regBody := jsonBody(t, models.RegisterRequest{
		Email:    "carol@example.com",
		Password: "securepass",
		Name:     "Carol",
		Role:     models.RoleCompany,
	})
	srv.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", regBody))

	// Now login
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		jsonBody(t, models.LoginRequest{Email: "carol@example.com", Password: "securepass"}))
	rec := httptest.NewRecorder()
	srv.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv := newTestServer(t)
	regBody := jsonBody(t, models.RegisterRequest{
		Email:    "dave@example.com",
		Password: "correctpass",
		Name:     "Dave",
		Role:     models.RoleStudent,
	})
	srv.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", regBody))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		jsonBody(t, models.LoginRequest{Email: "dave@example.com", Password: "wrongpass"}))
	rec := httptest.NewRecorder()
	srv.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
