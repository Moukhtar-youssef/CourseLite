package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

type Handler struct {
	DB            *sql.DB
	AccessSecret  string
	RefreshSecret string
}

// ── Register ────────────────────────────────────────────────────────────────

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if body.Email == "" || body.Password == "" || body.Name == "" {
		jsonError(w, "name, email and password are required", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check if email already exists
	var exists bool
	h.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, body.Email).Scan(&exists)
	if exists {
		jsonError(w, "email already in use", http.StatusConflict)
		return
	}

	hash, err := HashPassword(body.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	var userID string
	err = h.DB.QueryRow(
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id`,
		body.Name, body.Email, hash,
	).Scan(&userID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.issueTokens(w, userID, body.Email)
}

// ── Login ────────────────────────────────────────────────────────────────────

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	var userID, hash string
	err := h.DB.QueryRow(
		`SELECT id, password_hash FROM users WHERE email=$1`, body.Email,
	).Scan(&userID, &hash)

	// Same error for wrong email OR wrong password — prevents user enumeration
	if err != nil || !CheckPassword(body.Password, hash) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	h.issueTokens(w, userID, body.Email)
}

// ── Refresh ──────────────────────────────────────────────────────────────────

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Read refresh token from HttpOnly cookie only
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		jsonError(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	claims, err := VerifyToken(cookie.Value, h.RefreshSecret)
	if err != nil || claims.Type != "refresh" {
		jsonError(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Check token is still in DB (allows server-side revocation)
	var stored string
	err = h.DB.QueryRow(
		`SELECT token FROM refresh_tokens WHERE user_id=$1 AND token=$2 AND expires_at > NOW()`,
		claims.UserID, cookie.Value,
	).Scan(&stored)
	if err != nil {
		jsonError(w, "refresh token revoked", http.StatusUnauthorized)
		return
	}

	// Rotate: delete old, issue new
	h.DB.Exec(`DELETE FROM refresh_tokens WHERE token=$1`, cookie.Value)
	h.issueTokens(w, claims.UserID, claims.Email)
}

// ── Logout ───────────────────────────────────────────────────────────────────

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("refresh_token"); err == nil {
		h.DB.Exec(`DELETE FROM refresh_tokens WHERE token=$1`, c.Value)
	}
	clearCookie(w, "access_token")
	clearCookie(w, "refresh_token")
	w.WriteHeader(http.StatusNoContent)
}

// ── Password Reset ───────────────────────────────────────────────────────────

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Always return 200 — prevents email enumeration
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "if that email exists you will receive a reset link",
	})

	// Check user exists then generate token
	var userID string
	err := h.DB.QueryRow(`SELECT id FROM users WHERE email=$1`, body.Email).Scan(&userID)
	if err != nil {
		return // silently do nothing
	}

	token := randomHex(32)
	h.DB.Exec(
		`INSERT INTO password_reset_tokens (user_id, token, expires_at)
         VALUES ($1, $2, $3)
         ON CONFLICT (user_id) DO UPDATE SET token=$2, expires_at=$3`,
		userID, token, time.Now().Add(1*time.Hour),
	)

	// TODO: send email with link: https://yourdomain.com/reset-password?token=<token>
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	if len(body.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	var userID string
	err := h.DB.QueryRow(
		`SELECT user_id FROM password_reset_tokens
         WHERE token=$1 AND expires_at > NOW()`, body.Token,
	).Scan(&userID)
	if err != nil {
		jsonError(w, "invalid or expired token", http.StatusBadRequest)
		return
	}

	hash, _ := HashPassword(body.Password)
	h.DB.Exec(`UPDATE users SET password_hash=$1 WHERE id=$2`, hash, userID)
	h.DB.Exec(`DELETE FROM password_reset_tokens WHERE user_id=$1`, userID)

	// Revoke all refresh tokens on password reset — force re-login everywhere
	h.DB.Exec(`DELETE FROM refresh_tokens WHERE user_id=$1`, userID)

	json.NewEncoder(w).Encode(map[string]string{"message": "password updated"})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) issueTokens(w http.ResponseWriter, userID, email string) {
	accessToken, _ := NewAccessToken(userID, email, h.AccessSecret)
	refreshToken, _ := NewRefreshToken(userID, email, h.RefreshSecret)

	// Store refresh token in DB for revocation support
	h.DB.Exec(
		`INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)`,
		userID, refreshToken, time.Now().Add(7*24*time.Hour),
	)

	// Access token: HttpOnly, short-lived
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   60 * 15, // 15 minutes
	})

	// Refresh token: HttpOnly, long-lived
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/auth/refresh", // scoped — only sent on refresh endpoint
		MaxAge:   60 * 60 * 24 * 7,    // 7 days
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id": userID,
		"email":   email,
	})
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, MaxAge: -1, Path: "/"})
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
