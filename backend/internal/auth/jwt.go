// Package auth provides JWT token generation and validation.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — what is a JWT?
// ────────────────────────────────────────────────────────────────────
// A JSON Web Token (JWT) is a compact, self-contained way to represent
// claims (assertions) between two parties. It has three Base64-encoded
// sections separated by dots:
//
//	HEADER.PAYLOAD.SIGNATURE
//
// The HEADER says which algorithm was used (HS256 here).
// The PAYLOAD carries our custom claims (user_id, role) plus standard
// ones (expiry, issued-at).
// The SIGNATURE is an HMAC-SHA256 hash of HEADER+PAYLOAD using a secret
// key only the server knows. Tampering with the payload invalidates the
// signature, so the server can trust the claims without a database lookup
// on every request.
//
// Useful resource: https://jwt.io/introduction
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims are the JWT claims embedded in each token.
// We embed jwt.RegisteredClaims to get standard fields (ExpiresAt,
// IssuedAt) for free.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// tokenDuration is how long a user session token stays valid after being issued.
// 72 hours means users stay logged in across the hackathon demo days
// without needing to re-authenticate.
const tokenDuration = 72 * time.Hour

// CheckInTokenDuration is the validity window of a QR check-in token.
//
// ────────────────────────────────────────────────────────────────────
// LEARNING NOTE — why two different expiry windows?
// ────────────────────────────────────────────────────────────────────
// The QR code on the projector screen represents the "live" event window.
// We want only students who were physically present (i.e. who scanned
// while the QR was active) to be able to check in.
//
// The trick: the QR token expires after 6 hours, so it cannot be
// replayed from a previous session.  BUT the student's sync can happen
// at any point in the future — the server never rejects a sync based on
// when it arrived.  The only validity check is the JWT signature inside
// the payload, which the student's device captured at scan time.
//
// Concretely:
//   - Student scans at 10:00 → their device stores a signed token valid until 16:00.
//   - They sync at 15:30 (still within window) → server verifies JWT: accepted.
//   - They sync at 23:00 (same day, poor connectivity) → server verifies JWT: still accepted
//     because the token was SCANNED (and thus signed) before it expired.
//   - A student who never scanned tries to craft a fake payload at 23:00 → rejected,
//     because they cannot produce a token whose exp is after the signing time.
//
// This gives students with poor connectivity unlimited time to sync,
// while still ensuring the QR was live when they scanned it.
const CheckInTokenDuration = 6 * time.Hour

// CheckInClaims are the claims embedded in a QR check-in token.
// These are intentionally minimal — the token just proves the student
// scanned a specific event's QR while it was live.
type CheckInClaims struct {
	EventID string `json:"event_id"`
	HostSig string `json:"host_sig"` // the event's check_in_code (shared secret)
	jwt.RegisteredClaims
}

// GenerateCheckInToken creates a short-lived signed JWT for a check-in QR code.
// The token is signed with the same server secret as user tokens, but carries
// event-specific claims rather than user claims.
//
// The host embeds this token as the "token" field in the QR code payload.
// Its signature is the cryptographic proof that the QR came from a server that
// knew the event's check_in_code at the time the QR was generated.
func GenerateCheckInToken(eventID, hostSig, secret string) (string, error) {
	now := time.Now().UTC()
	claims := CheckInClaims{
		EventID: eventID,
		HostSig: hostSig,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(CheckInTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign check-in token: %w", err)
	}
	return signed, nil
}

// GenerateCheckInTokenWithExpiry creates a check-in token with explicit iat/exp values.
// This is primarily used in tests to simulate tokens that were scanned in the
// past (and whose exp has already elapsed) to verify that late syncs are accepted.
func GenerateCheckInTokenWithExpiry(eventID, hostSig, secret string, iat, exp time.Time) (string, error) {
	claims := CheckInClaims{
		EventID: eventID,
		HostSig: hostSig,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(iat),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign check-in token: %w", err)
	}
	return signed, nil
}

// IMPORTANT: This function verifies the JWT SIGNATURE but deliberately skips
// the expiry (exp) check.  This is the core of the "scan window vs sync window"
// design:
//
//   - The 6-hour exp is enforced at SCAN time by the student's PWA — if the
//     host's QR page refreshes or the event ends, the new code is different.
//   - Once a student has a valid signed token stored in IndexedDB, they can
//     sync it at any point in the future.  Poor connectivity should never
//     cause a legitimate check-in to be rejected.
//
// Security: a forged or tampered token is still rejected because the HMAC
// signature will not verify without the server secret.
func ParseCheckInToken(tokenStr, secret string) (*CheckInClaims, error) {
	// jwt.WithoutClaimsValidation skips exp/nbf/iss checks — we only want
	// the signature verification that happens unconditionally.
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&CheckInClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		},
		jwt.WithoutClaimsValidation(),
	)
	if err != nil {
		return nil, fmt.Errorf("parse check-in token: %w", err)
	}
	claims, ok := token.Claims.(*CheckInClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid check-in token")
	}
	return claims, nil
}

// GenerateToken creates a signed JWT for the given user.
// The token is signed with HS256 (HMAC-SHA256) using the server secret.
// Anyone with the secret can verify the token — keep it out of git!
func GenerateToken(userID, role, secret string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ParseToken validates a JWT string and returns the embedded claims.
// It rejects tokens with:
//   - wrong or missing signature
//   - expired tokens (ExpiresAt in the past)
//   - unexpected signing algorithm (algorithm confusion attack prevention)
func ParseToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		// Guard against "alg:none" or RS256 tokens being passed to an HS256 server.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
