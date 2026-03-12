package auth

import (
	"context"
	"errors"
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
				jsonError(w, "missing token", http.StatusUnauthorized)
				return
			}

			claims, err := VerifyToken(token, accessSecret)
			if err != nil {
				if errors.Is(err, ErrExpiredToken) {
					jsonError(w, "token expired", http.StatusUnauthorized)
					return
				}
				jsonError(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Reject refresh tokens — they must never access protected routes
			if claims.Type != "access" {
				jsonError(w, "invalid token type", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser pulls claims from context in any protected handler
func GetUser(r *http.Request) *Claims {
	claims, _ := r.Context().Value(UserClaimsKey).(*Claims)
	return claims
}

// extractToken checks Authorization header first, then HttpOnly cookie
func extractToken(r *http.Request) string {
	// Bearer token — for API / mobile clients
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	// HttpOnly cookie — for browser clients
	if c, err := r.Cookie("access_token"); err == nil {
		return c.Value
	}
	return ""
}
