// Package handlers contains HTTP handlers for the CourseLite service layer.
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Moukhtar-youssef/CourseLite/internal/auth"
)

// JSONError writes a JSON error response with the given message and status code.
func JSONError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// JSONResponse writes a JSON response with the given value and status code.
func JSONResponse(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// JSONMessage writes a JSON message response with the given message and status code.
func JSONMessage(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func clearCookie(w http.ResponseWriter, name string) {
	secure := false
	path := "/"
	if name == "refresh_token" {
		secure = true
		path = "/api/auth/refresh"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		HttpOnly: true,
		MaxAge:   -1,
		Secure:   secure,
		Path:     path,
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("randomHex: crypto/rand unavailable: %v", err))
	}
	return hex.EncodeToString(b)
}

func claimsFromCookie(r *http.Request, secret string) (auth.Claims, error) {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		return auth.Claims{}, err
	}
	claims, err := auth.VerifyToken(cookie.Value, secret)
	if err != nil || claims.Type != "access" {
		return auth.Claims{}, err
	}
	return claims, nil
}
