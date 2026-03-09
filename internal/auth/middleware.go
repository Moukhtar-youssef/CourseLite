package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

func Middleware(accessSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := VerifyToken(token, accessSecret)
			if err != nil {
				if err == ErrExpiredToken {
					http.Error(w, `{"error":"token expired"}`, http.StatusUnauthorized)
					return
				}
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// Only allow access tokens through — not refresh tokens
			if claims.Type != "access" {
				http.Error(w, `{"error":"invalid token type"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser pulls claims from context in any handler
func GetUser(r *http.Request) *Claims {
	claims, _ := r.Context().Value(UserClaimsKey).(*Claims)
	return claims
}

// Supports: Authorization: Bearer <token>
// Also checks HttpOnly cookie as fallback for browser clients
func extractToken(r *http.Request) string {
	// 1. Check Authorization header (API clients)
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// 2. Check HttpOnly cookie (browser clients)
	if c, err := r.Cookie("access_token"); err == nil {
		return c.Value
	}
	return ""
}
