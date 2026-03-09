package auth

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthHandler struct {
	Handler
	Config *oauth2.Config
}

func NewGoogleOAuth(
	db *sql.DB,
	accessSecret, refreshSecret, clientID, clientSecret, redirectURL string,
) *OAuthHandler {
	return &OAuthHandler{
		Handler: Handler{DB: db, AccessSecret: accessSecret, RefreshSecret: refreshSecret},
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (h *OAuthHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	state := randomHex(16) // TODO: store in session/cookie to verify on callback
	url := h.Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		jsonError(w, "missing code", http.StatusBadRequest)
		return
	}

	t, err := h.Config.Exchange(r.Context(), code)
	if err != nil {
		jsonError(w, "oauth exchange failed", http.StatusInternalServerError)
		return
	}

	// Fetch user info from Google
	client := h.Config.Client(r.Context(), t)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		jsonError(w, "failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&googleUser)

	// Upsert user — create if new, fetch if existing
	var userID string
	err = h.DB.QueryRow(`
        INSERT INTO users (name, email, oauth_provider, oauth_id)
        VALUES ($1, $2, 'google', $3)
        ON CONFLICT (email) DO UPDATE SET name=EXCLUDED.name
        RETURNING id
    `, googleUser.Name, googleUser.Email, googleUser.ID).Scan(&userID)
	if err != nil {
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.issueTokens(w, userID, googleUser.Email)
	http.Redirect(w, r, "/dashboard", http.StatusTemporaryRedirect)
}
