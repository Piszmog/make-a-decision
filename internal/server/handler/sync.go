package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Piszmog/make-a-decision/internal/db/queries"
	"github.com/Piszmog/make-a-decision/internal/server/utils"
)

type LocalOption struct {
	Text     string   `json:"text"`
	Weight   int64    `json:"weight"`
	Duration *int64   `json:"duration"`
	Tags     []string `json:"tags"`
}

type SyncRequest struct {
	Options []LocalOption `json:"options"`
}

type SyncResponse struct {
	Success bool   `json:"success"`
	Synced  int    `json:"synced"`
	Message string `json:"message,omitempty"`
}

func (h *Handler) SyncLocalOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	syncedCount := 0

	// Create each option
	for _, opt := range req.Options {
		// Validate
		if strings.TrimSpace(opt.Text) == "" {
			h.Logger.Warn("Skipping option with empty text during sync")
			continue
		}

		// Clamp weight to valid range (1-10)
		weight := max(1, min(opt.Weight, 10))

		// Validate duration
		var durationParam any
		if opt.Duration != nil {
			dur := *opt.Duration
			if dur < 0 || dur > 1440 {
				h.Logger.Warn("Invalid duration during sync, skipping", "duration", dur)
				continue
			}
			durationParam = dur
		}

		// Create option
		createParams := queries.CreateOptionParams{
			Name:            opt.Text,
			Weight:          sql.NullInt64{Int64: weight, Valid: true},
			DurationMinutes: durationParam,
			UserID:          userID,
		}

		createdOption, err := h.Database.Queries().CreateOption(ctx, createParams)
		if err != nil {
			h.Logger.Error("Failed to sync option", "error", err, "option", opt.Text)
			continue
		}

		// Add tags (limit to 5)
		tags := opt.Tags
		if len(tags) > 5 {
			tags = tags[:5]
		}

		if len(tags) > 0 {
			if err := h.setTagsForOption(ctx, createdOption.ID, userID, tags); err != nil {
				h.Logger.Warn("Failed to sync tags for option", "error", err, "option_id", createdOption.ID)
			}
		}

		syncedCount++
	}

	h.Logger.Info("Local options synced", "user_id", userID, "count", syncedCount)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(SyncResponse{
		Success: true,
		Synced:  syncedCount,
		Message: "Options synced successfully",
	}); err != nil {
		h.Logger.Error("Failed to encode sync response", "error", err)
	}
}
