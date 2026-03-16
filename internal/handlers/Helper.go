// Package handlers is a package that contain the handlers for several services
// e.g. database
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func stringToUUID(s string) (uuid.UUID, error) {
	UUID, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, err
	}
	return UUID, nil
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
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
	rand.Read(b)
	return hex.EncodeToString(b)
}
