package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Piszmog/make-a-decision/internal/db"
	"github.com/a-h/templ"
)

// Handler handles requests.
type Handler struct {
	Logger   *slog.Logger
	Database db.Database
}

//nolint:unparam
func (h *Handler) html(ctx context.Context, w http.ResponseWriter, status int, t templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := t.Render(ctx, w); err != nil {
		h.Logger.Error("Failed to render component", "error", err)
	}
}
