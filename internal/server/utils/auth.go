package utils

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const userIDContextKey contextKey = "user_id"

// CheckPasswordHash compares a bcrypt hashed password with plaintext
func CheckPasswordHash(hash, password []byte) error {
	return bcrypt.CompareHashAndPassword(hash, password)
}

// SetSessionCookie sets the session cookie with proper security settings
func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

// GetClientIP extracts the client IP from the request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// GetUserEmail extracts the user email from request header (set by middleware)
func GetUserEmail(r *http.Request) string {
	return r.Header.Get("USER-EMAIL")
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value(userIDContextKey).(int64)
	return userID, ok
}

// RequireAuth checks for authenticated user and returns user ID
// Returns (0, false) and sends HTTP error if not authenticated
func RequireAuth(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return 0, false
	}
	return userID, true
}

// SetUserID adds user ID to request context
func SetUserID(r *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(r.Context(), userIDContextKey, userID)
	return r.WithContext(ctx)
}
