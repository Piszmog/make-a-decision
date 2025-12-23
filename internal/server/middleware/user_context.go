package middleware

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Piszmog/make-a-decision/internal/db"
	"github.com/Piszmog/make-a-decision/internal/db/queries"
	"github.com/Piszmog/make-a-decision/internal/server/utils"
)

const (
	SessionDuration      = 7 * 24 * time.Hour
	sessionRefreshWindow = 24 * time.Hour
)

type UserContextMiddleware struct {
	Logger   *slog.Logger
	Database db.Database
}

// Middleware adds user context to requests if they have a valid session
// Routes remain public - this just enriches the request with user info
func (m *UserContextMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			// No session cookie - user is not signed in, continue anyway
			next.ServeHTTP(w, r)
			return
		}

		session, err := m.Database.Queries().GetSessionByToken(r.Context(), cookie.Value)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				m.Logger.ErrorContext(r.Context(), "failed to get session", "err", err)
			}
			// Invalid/expired session - clear cookie and continue
			utils.ClearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		// Check if session expired
		if session.ExpiresAt.Before(time.Now()) {
			m.Logger.DebugContext(r.Context(), "session expired", "session_id", session.ID)
			_ = m.Database.Queries().DeleteSessionByToken(r.Context(), cookie.Value)
			utils.ClearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}

		// Session is valid - refresh if needed
		timeUntilExpiry := time.Until(session.ExpiresAt)
		if timeUntilExpiry > 0 && timeUntilExpiry < sessionRefreshWindow {
			m.Logger.DebugContext(r.Context(), "refreshing session", "user_id", session.UserID)
			newExpiry := time.Now().Add(SessionDuration)
			err = m.Database.Queries().UpdateSessionExpiresAt(r.Context(), queries.UpdateSessionExpiresAtParams{
				Token:     session.Token,
				ExpiresAt: newExpiry,
			})
			if err != nil {
				m.Logger.ErrorContext(r.Context(), "failed to refresh session", "err", err)
			} else {
				utils.SetSessionCookie(w, session.Token, newExpiry)
			}
		}

		// Get user info
		user, err := m.Database.Queries().GetUserByID(r.Context(), session.UserID)
		if err != nil {
			m.Logger.ErrorContext(r.Context(), "failed to get user", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		// Add user email to request header for templates
		r.Header.Set("USER-EMAIL", user.Email)

		// Add user ID to request context
		r = utils.SetUserID(r, user.ID)

		next.ServeHTTP(w, r)
	})
}
