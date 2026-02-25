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

// tokenDuration is how long a token stays valid after being issued.
// 72 hours means users stay logged in across the hackathon demo days
// without needing to re-authenticate.
const tokenDuration = 72 * time.Hour

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
