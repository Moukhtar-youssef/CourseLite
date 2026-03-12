package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"time"

	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	DB            *DB.Queries
	AccessSecret  string
	RefreshSecret string
}

// ── pgtype conversion helpers ─────────────────────────────────────────────────

func toPgtypeText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	)
}

func stringToUUID(s string) (uuid.UUID, error) {
	UUID, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, err
	}
	return UUID, nil
}

// ── Register ──────────────────────────────────────────────────────────────────

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Name == "" || body.Email == "" || body.Password == "" {
		jsonError(w, "name, email and password are required", http.StatusBadRequest)
		return
	}

	if _, err := mail.ParseAddress(body.Email); err != nil {
		jsonError(w, "invalid email address", http.StatusBadRequest)
		return
	}

	if len(body.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	exists, err := h.DB.EmailExists(r.Context(), body.Email)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	if exists {
		jsonError(w, "email already in use", http.StatusConflict)
		return
	}

	hash, err := HashPassword(body.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.DB.CreateUser(r.Context(), DB.CreateUserParams{
		Name:         body.Name,
		Email:        body.Email,
		PasswordHash: toPgtypeText(hash),
	})
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.issueTokens(w, r.Context(), user.ID.String(), user.Email)
}

// ── Login ─────────────────────────────────────────────────────────────────────

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Email == "" || body.Password == "" {
		jsonError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.DB.GetUserByEmail(r.Context(), body.Email)
	// Same error for wrong email OR wrong password — prevents user enumeration
	if err != nil || !CheckPassword(body.Password, user.PasswordHash.String) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	h.issueTokens(w, r.Context(), user.ID.String(), user.Email)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
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

	ClaimsUserID, err := stringToUUID(claims.UserID)
	if err != nil {
		jsonError(w, "invalid uuid", http.StatusUnauthorized)
		return
	}
	valid, err := h.DB.RefreshTokenExists(r.Context(), DB.RefreshTokenExistsParams{
		UserID: ClaimsUserID,
		Token:  cookie.Value,
	})
	if err != nil || !valid {
		jsonError(w, "refresh token revoked", http.StatusUnauthorized)
		return
	}

	h.DB.DeleteRefreshToken(r.Context(), cookie.Value)
	h.issueTokens(w, r.Context(), claims.UserID, claims.Email)
}

// ── Logout ────────────────────────────────────────────────────────────────────

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("refresh_token"); err == nil {
		h.DB.DeleteRefreshToken(r.Context(), c.Value)
	}
	clearCookie(w, "access_token")
	clearCookie(w, "refresh_token")
	w.WriteHeader(http.StatusNoContent)
}

// ── Forgot Password ───────────────────────────────────────────────────────────

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Always respond 200 — prevents email enumeration
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "if that email exists you will receive a reset link",
	})

	user, err := h.DB.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		return // user doesn't exist — silently do nothing
	}

	token := randomHex(32)
	h.DB.UpsertPasswordResetToken(r.Context(), DB.UpsertPasswordResetTokenParams{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})

	// TODO: send email — https://domain/reset-password?token=<token>
}

// ── Reset Password ────────────────────────────────────────────────────────────

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	userID, err := h.DB.GetUserIDByResetToken(r.Context(), body.Token)
	if err != nil {
		jsonError(w, "invalid or expired token", http.StatusBadRequest)
		return
	}

	hash, err := HashPassword(body.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.DB.UpdateUserPassword(r.Context(), DB.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: toPgtypeText(hash),
	})
	h.DB.DeletePasswordResetToken(r.Context(), userID)
	h.DB.DeleteAllRefreshTokens(r.Context(), userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "password updated"})
}

// ── Token issuance ────────────────────────────────────────────────────────────

func (h *Handler) issueTokens(w http.ResponseWriter, ctx context.Context, userID, email string) {
	accessToken, err := NewAccessToken(userID, email, h.AccessSecret)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	refreshToken, err := NewRefreshToken(userID, email, h.RefreshSecret)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	UserUUID, err := stringToUUID(userID)
	if err != nil {
		jsonError(w, "invalid uuid", http.StatusUnauthorized)
		return
	}

	h.DB.CreateRefreshToken(ctx, DB.CreateRefreshTokenParams{
		UserID:    UserUUID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   60 * 15,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api/auth/refresh",
		MaxAge:   60 * 60 * 24 * 7,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id": userID,
		"email":   email,
	})
}

// ── Low-level helpers ─────────────────────────────────────────────────────────

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
