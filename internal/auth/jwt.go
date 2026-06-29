// Package auth provides password hashing and verification using Argon2id and Token Handling
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

// Claims represents the JWT token payload containing user identity
// information and standard JWT claims. It is used for both access
// and refresh tokens.
type Claims struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	ProfilePic string `json:"profilePic"`
	Role       string `json:"role"`
	Type       string `json:"type"`
	jwt.RegisteredClaims
}

// NewAccessToken generates a new JWT access token for the given user.
// It takes userID, email, and secret as parameters and returns the token string and any error.
// The token expires after 15 minutes.
func NewAccessToken(userID, email, profilePic, role, secret string) (string, error) {
	if profilePic == "" {
		profilePic = "Placeholder for the normal userpic"
	}
	claims := Claims{
		UserID:     userID,
		Email:      email,
		ProfilePic: profilePic,
		Role:       role,
		Type:       "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "courselite",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(secret))
}

// NewRefreshToken generates a new JWT refresh token for the given user.
// It takes userID, email, and secret as parameters and returns the token string,
// the JWT ID (jti), and any error. The token expires after 7 days.
func NewRefreshToken(userID, email, profilePic, role, secret string) (string, string, error) {
	jti := uuid.NewString()
	if profilePic == "" {
		profilePic = "Placeholder for the normal userpic"
	}

	claims := Claims{
		UserID:     userID,
		Email:      email,
		ProfilePic: profilePic,
		Role:       role,
		Type:       "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    "courselite",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(secret))

	return token, jti, err
}

// VerifyToken validates a JWT token string using the provided secret.
// It returns the claims if the token is valid, or an error if the token
// is invalid or expired.
func VerifyToken(tokenStr, secret string) (Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}

			return []byte(secret), nil
		},
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return Claims{}, ErrExpiredToken
		}
		return Claims{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(Claims)

	if !ok || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	return claims, nil
}

// HashToken generates a SHA-256 hash of the given token string.
// It returns the hexadecimal encoded hash string.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
