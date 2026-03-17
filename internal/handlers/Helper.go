// Package handlers is a package that contain the handlers for several services
// e.g. database
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func stringToUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("stringToUUID: invalid UUID %q: %w", s, err)
	}
	return id, nil
}

func JsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func JsonResponse(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func JsonMessage(w http.ResponseWriter, msg string, status int) {
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
