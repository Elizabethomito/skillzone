package auth

import (
	"testing"
	"time"
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

// ---- Check-in token tests ----

func TestGenerateAndParseCheckInToken(t *testing.T) {
	eventID := "event-abc"
	hostSig := "secret-code-xyz"

	token, err := GenerateCheckInToken(eventID, hostSig, testSecret)
	if err != nil {
		t.Fatalf("GenerateCheckInToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty check-in token")
	}

	claims, err := ParseCheckInToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseCheckInToken: %v", err)
	}
	if claims.EventID != eventID {
		t.Errorf("EventID: got %q, want %q", claims.EventID, eventID)
	}
	if claims.HostSig != hostSig {
		t.Errorf("HostSig: got %q, want %q", claims.HostSig, hostSig)
	}
}

// TestParseCheckInToken_AcceptsExpiredToken verifies the core design property:
// ParseCheckInToken accepts a token whose exp is in the past.
// The exp only controlled the scan window; sync is allowed at any later time.
func TestParseCheckInToken_AcceptsExpiredToken(t *testing.T) {
	iat := time.Now().Add(-7 * 24 * time.Hour) // scanned 1 week ago
	exp := iat.Add(CheckInTokenDuration)       // expired ~6h after scanning

	token, err := GenerateCheckInTokenWithExpiry("event-xyz", "sig", testSecret, iat, exp)
	if err != nil {
		t.Fatalf("GenerateCheckInTokenWithExpiry: %v", err)
	}

	// Must NOT return an error despite exp being in the past.
	claims, err := ParseCheckInToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseCheckInToken rejected a late-synced token: %v", err)
	}
	if claims.EventID != "event-xyz" {
		t.Errorf("EventID: got %q, want %q", claims.EventID, "event-xyz")
	}
}

// TestParseCheckInToken_RejectsWrongSecret verifies that a token signed by a
// different server secret is rejected even if the exp is still valid.
func TestParseCheckInToken_RejectsWrongSecret(t *testing.T) {
	token, err := GenerateCheckInToken("event-abc", "sig", testSecret)
	if err != nil {
		t.Fatalf("GenerateCheckInToken: %v", err)
	}
	_, err = ParseCheckInToken(token, "different-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

// TestParseCheckInToken_RejectsTamperedPayload verifies that modifying the
// token string causes a signature verification failure.
func TestParseCheckInToken_RejectsTamperedPayload(t *testing.T) {
	_, err := ParseCheckInToken("eyJhbGciOiJIUzI1NiJ9.eyJldmVudF9pZCI6ImZha2UifQ.invalidsig", testSecret)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}
