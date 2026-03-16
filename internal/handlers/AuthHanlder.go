// Package handlers is a package that contain the handlers for several services
// e.g. database
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/mail"
	"time"

	"github.com/Moukhtar-youssef/CourseLite/internal/auth"
	DB "github.com/Moukhtar-youssef/CourseLite/internal/db"
	"github.com/google/uuid"
)

type AuthHandler struct {
	DB            *DB.Queries
	AccessSecret  string
	RefreshSecret string
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request", 400)
		return
	}

	if body.Name == "" || body.Email == "" || body.Password == "" {
		jsonError(w, "missing fields", 400)
		return
	}

	if _, err := mail.ParseAddress(body.Email); err != nil {
		jsonError(w, "invalid email", 400)
		return
	}

	exists, _ := h.DB.EmailExists(r.Context(), body.Email)

	if exists {
		jsonError(w, "email already used", 409)
		return
	}

	hash, _ := auth.HashPassword(body.Password)

	user, err := h.DB.CreateUser(r.Context(), DB.CreateUserParams{
		Name:         body.Name,
		Email:        body.Email,
		PasswordHash: &hash,
	})
	if err != nil {
		jsonError(w, "server error", 500)
		return
	}

	h.issueTokens(w, r.Context(), user.ID.String(), user.Email,
		r.RemoteAddr, r.UserAgent())
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
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
	// Same error for wrong email OR wrong password to prevents user enumeration
	if err != nil || !auth.CheckPassword(body.Password, *user.PasswordHash) {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	h.issueTokens(w, r.Context(), user.ID.String(), user.Email, r.RemoteAddr,
		r.UserAgent())
}

func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		jsonError(w, "missing access token", http.StatusUnauthorized)
		return
	}

	claims, err := auth.VerifyToken(cookie.Value, h.AccessSecret)
	if err != nil || claims.Type != "access" {
		jsonError(w, "invalid token", http.StatusUnauthorized)
		return
	}
	ClaimsUserID, err := stringToUUID(claims.UserID)
	if err != nil {
		jsonError(w, "invalid uuid", http.StatusUnauthorized)
		return
	}

	sessions, err := h.DB.GetUserSessions(r.Context(), ClaimsUserID)
	if err != nil {
		jsonError(w, "server error", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sessions)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		jsonError(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	claims, err := auth.VerifyToken(cookie.Value, h.RefreshSecret)
	if err != nil || claims.Type != "refresh" {
		jsonError(w, "invalid token", http.StatusUnauthorized)
		return
	}

	ClaimsUserID, err := stringToUUID(claims.UserID)
	if err != nil {
		jsonError(w, "invalid uuid", http.StatusUnauthorized)
		return
	}
	hash := auth.HashToken(cookie.Value)
	valid, err := h.DB.RefreshTokenExists(r.Context(),
		DB.RefreshTokenExistsParams{
			UserID:    ClaimsUserID,
			TokenHash: hash,
		})
	if err != nil {
		jsonError(w, "Server Error", http.StatusInternalServerError)
		return
	}

	if !valid {
		h.DB.DeleteAllRefreshTokens(r.Context(), ClaimsUserID)

		jsonError(w, "Session compromised", http.StatusUnauthorized)
		return
	}

	h.DB.DeleteRefreshToken(r.Context(), hash)
	h.issueTokens(w, r.Context(), claims.UserID, claims.Email, r.RemoteAddr,
		r.UserAgent())
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("refresh_token"); err == nil {
		h.DB.DeleteRefreshToken(r.Context(), auth.HashToken(c.Value))
	}
	clearCookie(w, "access_token")
	clearCookie(w, "refresh_token")
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "if that email exists you will receive a reset link",
	})

	user, err := h.DB.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		return // user doesn't exist then silently do nothing
	}

	token := randomHex(32)
	h.DB.UpsertPasswordResetToken(r.Context(), DB.UpsertPasswordResetTokenParams{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	// TODO: send email — https://domain/reset-password?token=<token>
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 8 {
		jsonError(w, "password must be at least 8 characters",
			http.StatusBadRequest)
		return
	}

	userID, err := h.DB.GetUserIDByResetToken(r.Context(), body.Token)
	if err != nil {
		jsonError(w, "invalid or expired token", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.DB.UpdateUserPassword(r.Context(), DB.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: &hash,
	})
	h.DB.DeletePasswordResetToken(r.Context(), userID)
	h.DB.DeleteAllRefreshTokens(r.Context(), userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "password updated"})
}

func (h *AuthHandler) issueTokens(w http.ResponseWriter, ctx context.Context,
	userID, email, ip, userAgent string,
) {
	accessToken, err := auth.NewAccessToken(userID, email, h.AccessSecret)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}
	refreshToken, jti, err := auth.NewRefreshToken(userID, email, h.RefreshSecret)
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
		TokenID:   uuid.MustParse(jti),
		UserID:    UserUUID,
		TokenHash: auth.HashToken(refreshToken),
		IpAddress: &ip,
		UserAgent: &userAgent,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   60 * 15,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/api/auth/refresh",
		MaxAge:   60 * 60 * 24 * 7,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id": userID,
		"email":   email,
	})
}
