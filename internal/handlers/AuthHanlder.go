// Package handlers contains HTTP handlers for the CourseLite service layer.
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

// AuthHandler handles all authentication-related HTTP routes.
type AuthHandler struct {
	DB            *DB.Queries
	AccessSecret  string
	RefreshSecret string
}

// Register handles user registration by validating the request body,
// checking for existing email, hashing the password, creating the user,
// and issuing access and refresh tokens.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		ReType   string `json:"reType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.Email == "" || body.Password == "" {
		JSONError(w, "name, email, and password are required",
			http.StatusBadRequest)
		return
	}
	if body.Password != body.ReType {
		JSONError(w, "retyped password isn't a match to the password",
			http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(body.Email); err != nil {
		JSONError(w, "invalid email address", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 8 {
		JSONError(w, "password must be at least 8 characters",
			http.StatusBadRequest)
		return
	}

	exists, _ := h.DB.EmailExists(r.Context(), body.Email)
	if exists {
		JSONError(w, "email already in use", http.StatusConflict)
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		JSONError(w, "internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.DB.CreateUser(r.Context(), DB.CreateUserParams{
		Name:         body.Name,
		Email:        body.Email,
		PasswordHash: &hash,
	})
	if err != nil {
		JSONError(w, "could not create user", http.StatusInternalServerError)
		return
	}

	h.issueTokens(r.Context(), w, user.ID.String(), user.Email, user.Userpfpurl, user.Role,
		r.RemoteAddr, r.UserAgent())
}

// Login handles user authentication by validating credentials,
// verifying the password, and issuing access and refresh tokens.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Email == "" || body.Password == "" {
		JSONError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	user, err := h.DB.GetUserByEmail(r.Context(), body.Email)
	passwordChecked, PasswordCheckerr := auth.CheckPassword(body.Password, *user.PasswordHash)
	if err != nil || !passwordChecked || PasswordCheckerr != nil {
		JSONError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	h.issueTokens(r.Context(), w, user.ID.String(), user.Email, user.Userpfpurl, user.Role,
		r.RemoteAddr, r.UserAgent())
}

// Session retrieves the current user's active sessions by validating
// the access token and querying the database for user sessions.
func (h *AuthHandler) Session(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		JSONError(w, "missing access token", http.StatusUnauthorized)
		return
	}

	claims, err := auth.VerifyToken(cookie.Value, h.AccessSecret)
	if err != nil || claims.Type != "access" {
		JSONError(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	claimsUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		JSONError(w, "invalid token subject", http.StatusUnauthorized)
		return
	}

	sessions, err := h.DB.GetUserSessions(r.Context(), claimsUserID)
	if err != nil {
		JSONError(w, "could not retrieve sessions", http.StatusInternalServerError)
		return
	}

	JSONResponse(w, sessions, http.StatusOK)
}

// Refresh handles token refresh by validating the refresh token,
// checking its existence in the database, and issuing new access
// and refresh tokens.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		JSONError(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	claims, err := auth.VerifyToken(cookie.Value, h.RefreshSecret)
	if err != nil || claims.Type != "refresh" {
		JSONError(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	claimsUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		JSONError(w, "invalid token subject", http.StatusUnauthorized)
		return
	}

	hash := auth.HashToken(cookie.Value)
	valid, err := h.DB.RefreshTokenExists(r.Context(),
		DB.RefreshTokenExistsParams{
			UserID:    claimsUserID,
			TokenHash: hash,
		})
	if err != nil {
		JSONError(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !valid {
		_ = h.DB.DeleteAllRefreshTokens(r.Context(), claimsUserID)
		JSONError(w, "session compromised, please log in again",
			http.StatusUnauthorized)
		return
	}

	_ = h.DB.DeleteRefreshToken(r.Context(), hash)
	h.issueTokens(r.Context(), w, claims.UserID, claims.Email, claims.ProfilePic, claims.Role,
		r.RemoteAddr, r.UserAgent())
}

// Logout handles user logout by deleting the refresh token from the
// database and clearing the access and refresh token cookies.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("refresh_token"); err == nil {
		_ = h.DB.DeleteRefreshToken(r.Context(), auth.HashToken(c.Value))
	}
	clearCookie(w, "access_token")
	clearCookie(w, "refresh_token")
	w.WriteHeader(http.StatusNoContent)
}

// ForgotPassword handles password reset requests by validating the email,
// checking if a user exists, and generating a password reset token.
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	JSONMessage(w, "if that email exists you will receive a reset link",
		http.StatusOK)

	user, err := h.DB.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		return
	}

	token := randomHex(32)
	_ = h.DB.UpsertPasswordResetToken(r.Context(),
		DB.UpsertPasswordResetTokenParams{
			UserID:    user.ID,
			Token:     token,
			ExpiresAt: time.Now().Add(time.Hour),
		})

	// TODO: send email https://domain/reset-password?token=<token>
}

// ResetPassword handles password reset by validating the reset token,
// hashing the new password, updating the user's password, and
// invalidating all existing refresh tokens.
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	// TODO: convert this to a transaction
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.Token == "" {
		JSONError(w, "reset token is required", http.StatusBadRequest)
		return
	}
	if len(body.Password) < 8 {
		JSONError(w, "password must be at least 8 characters",
			http.StatusBadRequest)
		return
	}

	userID, err := h.DB.GetUserIDByResetToken(r.Context(), body.Token)
	if err != nil {
		JSONError(w, "invalid or expired reset token", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		JSONError(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = h.DB.UpdateUserPassword(r.Context(), DB.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: &hash,
	})
	_ = h.DB.DeletePasswordResetToken(r.Context(), userID)
	_ = h.DB.DeleteAllRefreshTokens(r.Context(), userID)

	JSONMessage(w, "password updated successfully", http.StatusOK)
}

func (h *AuthHandler) issueTokens(
	ctx context.Context, w http.ResponseWriter,
	userID, email, profilePic, role, ip, userAgent string,
) {
	accessToken, err := auth.NewAccessToken(userID, email, profilePic, role, h.AccessSecret)
	if err != nil {
		JSONError(w, "internal error", http.StatusInternalServerError)
		return
	}
	refreshToken, jti, err := auth.NewRefreshToken(userID, email, profilePic, role, h.RefreshSecret)
	if err != nil {
		JSONError(w, "internal error", http.StatusInternalServerError)
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		JSONError(w, "internal error", http.StatusInternalServerError)
		return
	}

	_ = h.DB.CreateRefreshToken(ctx, DB.CreateRefreshTokenParams{
		TokenID:   uuid.MustParse(jti),
		UserID:    userUUID,
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
		MaxAge:   15 * 60,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/api/auth/refresh",
		MaxAge:   7 * 24 * 60 * 60,
	})

	JSONResponse(w, map[string]string{
		"user_id": userID,
		"email":   email,
	}, http.StatusOK)
}
