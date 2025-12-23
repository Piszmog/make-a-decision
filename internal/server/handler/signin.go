package handler

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/Piszmog/make-a-decision/internal/components/auth"
	"github.com/Piszmog/make-a-decision/internal/db/queries"
	"github.com/Piszmog/make-a-decision/internal/server/middleware"
	"github.com/Piszmog/make-a-decision/internal/server/utils"
	"github.com/google/uuid"
)

// SigninPage renders the signin form
func (h *Handler) SigninPage(w http.ResponseWriter, r *http.Request) {
	utils.ClearSessionCookie(w)
	h.html(r.Context(), w, http.StatusOK, auth.SigninPage())
}

// Authenticate handles signin form submission
func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate inputs
	if email == "" || password == "" {
		h.Logger.DebugContext(ctx, "missing required form values", "email", email)
		h.html(ctx, w, http.StatusOK, auth.Alert("error", "Missing email or password", "Please enter your email and password."))
		return
	}

	// Get user by email
	user, err := h.Database.Queries().GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.Logger.DebugContext(ctx, "user not found", "email", email)
			h.html(ctx, w, http.StatusOK, auth.Alert("error", "Incorrect email or password", "Double check your email and password and try again."))
		} else {
			h.Logger.ErrorContext(ctx, "failed to get user", "error", err)
			h.html(ctx, w, http.StatusOK, auth.Alert("warning", "Something went wrong", "Try again later."))
		}
		return
	}

	// Verify password
	if err = utils.CheckPasswordHash([]byte(user.PasswordHash), []byte(password)); err != nil {
		h.Logger.DebugContext(ctx, "failed to compare password and hash", "error", err)
		h.html(ctx, w, http.StatusOK, auth.Alert("error", "Incorrect email or password", "Double check your email and password and try again."))
		return
	}

	// Check for existing session cookie
	var cookieValue string
	cookie, err := r.Cookie("session")
	if err == nil {
		cookieValue = cookie.Value
	}

	// Create new session
	token, expiresAt, err := h.newSession(ctx, user.ID, r.UserAgent(), cookieValue, utils.GetClientIP(r))
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to create session", "error", err)
		h.html(ctx, w, http.StatusOK, auth.Alert("warning", "Something went wrong", "Try again later."))
		return
	}

	utils.SetSessionCookie(w, token, expiresAt)

	// Clean up old sessions (7+ days old)
	err = h.Database.Queries().DeleteOldUserSessions(ctx, user.ID)
	if err != nil {
		h.Logger.WarnContext(ctx, "failed to delete old user sessions", "userID", user.ID, "error", err)
	}

	h.Logger.InfoContext(ctx, "User signed in successfully", "user_id", user.ID, "email", user.Email)

	// Trigger local storage sync on frontend
	w.Header().Set("HX-Trigger", `{"syncLocalStorage": true}`)

	// Redirect to home page
	w.Header().Set("HX-Redirect", "/")
}

// newSession creates a new session for the user
func (h *Handler) newSession(ctx context.Context, userID int64, userAgent string, currentToken string, ipAddress string) (string, time.Time, error) {
	// Delete old session if exists
	if currentToken != "" {
		if err := h.Database.Queries().DeleteSessionByToken(ctx, currentToken); err != nil {
			h.Logger.WarnContext(ctx, "failed to delete old session", "error", err)
		}
	}

	token := uuid.New().String()
	expiresAt := time.Now().Add(middleware.SessionDuration)

	session := queries.InsertSessionParams{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		UserAgent: sql.NullString{String: userAgent, Valid: userAgent != ""},
		IpAddress: sql.NullString{String: ipAddress, Valid: ipAddress != ""},
	}

	if err := h.Database.Queries().InsertSession(ctx, session); err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

// Signout handles user logout
func (h *Handler) Signout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		_ = h.Database.Queries().DeleteSessionByToken(r.Context(), cookie.Value)
	}

	utils.ClearSessionCookie(w)
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}
