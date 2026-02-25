package auth

import (
	"testing"
)

const testSecret = "super-secret-test-key"

func TestGenerateAndParseToken(t *testing.T) {
	userID := "user-123"
	role := "student"

	token, err := GenerateToken(userID, role, testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID: got %q, want %q", claims.UserID, userID)
	}
	if claims.Role != role {
		t.Errorf("Role: got %q, want %q", claims.Role, role)
	}
}

func TestParseToken_InvalidSecret(t *testing.T) {
	token, err := GenerateToken("user-abc", "company", testSecret)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	_, err = ParseToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for invalid secret, got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := ParseToken("not.a.real.token", testSecret)
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}
