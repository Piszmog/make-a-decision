package handler

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/Piszmog/make-a-decision/internal/components/core"
	"github.com/Piszmog/make-a-decision/internal/components/home"
	"github.com/Piszmog/make-a-decision/internal/db/queries"
	"github.com/Piszmog/make-a-decision/internal/server/utils"
)

// dbOptionToAppOption converts SQLC home.Option to app home.Option
func (h *Handler) dbOptionToAppOption(ctx context.Context, dbOpt queries.Option, userID int64) home.Option {
	var duration *int64
	if dbOpt.DurationMinutes != nil {
		if dur, ok := dbOpt.DurationMinutes.(int64); ok {
			duration = &dur
		}
	}

	var weight int64 = 1
	if dbOpt.Weight.Valid {
		weight = dbOpt.Weight.Int64
	}

	// Fetch tags for this option
	tags, err := h.fetchTagsForOption(ctx, dbOpt.ID, userID)
	if err != nil {
		// Log error but continue - tags are optional
		h.Logger.Warn("Failed to fetch tags for option", "option_id", dbOpt.ID, "error", err)
		tags = []string{}
	}

	return home.Option{
		ID:       strconv.FormatInt(dbOpt.ID, 10),
		Text:     dbOpt.Name,
		Weight:   weight,
		Duration: duration,
		Tags:     tags,
	}
}

// stringToInt64 converts string ID to int64 with error handling
func stringToInt64(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

// fetchTagsForOption retrieves all tags for a given option
func (h *Handler) fetchTagsForOption(ctx context.Context, optionID, userID int64) ([]string, error) {
	tags, err := h.Database.Queries().GetTagsForOption(ctx, queries.GetTagsForOptionParams{
		OptionID: optionID,
		UserID:   userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}

	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Name
	}
	return tagNames, nil
}

// setTagsForOption replaces all tags for an option
func (h *Handler) setTagsForOption(ctx context.Context, optionID, userID int64, tagNames []string) error {
	// Clear existing tags
	if err := h.Database.Queries().ClearTagsForOption(ctx, optionID); err != nil {
		return fmt.Errorf("failed to clear tags: %w", err)
	}

	// Add new tags
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(strings.ToLower(tagName))
		if tagName == "" {
			continue // Skip blank tags
		}

		// Get or create tag
		tag, err := h.Database.Queries().GetOrCreateTag(ctx, queries.GetOrCreateTagParams{
			LOWER:  tagName,
			UserID: userID,
		})
		if err != nil {
			return fmt.Errorf("failed to get/create tag %q: %w", tagName, err)
		}

		// Link tag to option
		if err := h.Database.Queries().AddTagToOption(ctx, queries.AddTagToOptionParams{
			OptionID: optionID,
			TagID:    tag.ID,
		}); err != nil {
			return fmt.Errorf("failed to add tag to option: %w", err)
		}
	}

	// Clean up unused tags
	if err := h.Database.Queries().DeleteUnusedTags(ctx, userID); err != nil {
		h.Logger.Warn("Failed to delete unused tags", "error", err)
	}

	return nil
}

// parseTagsFromForm parses comma-separated tags from form input
func parseTagsFromForm(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	tags := make([]string, 0, len(parts))

	for _, part := range parts {
		tag := strings.TrimSpace(strings.ToLower(part))
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	// Limit to 5 tags
	if len(tags) > 5 {
		tags = tags[:5]
	}

	return tags
}

// selectRandomOption implements weighted random selection from database with optional time constraint and tag filtering
func (h *Handler) selectRandomOption(ctx context.Context, userID int64, timeConstraintMinutes *int64, selectedTags []string) (home.Option, bool, error) {
	options, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		return home.Option{}, false, err
	}

	if len(options) == 0 {
		return home.Option{ID: "", Text: "No options available", Weight: 1}, false, nil
	}

	// Filter options by time constraint and tags if provided
	//nolint:prealloc
	var eligibleOptions []queries.Option
	for _, opt := range options {
		// Time constraint filter
		// Always include options without a duration (nil duration means flexible)
		if timeConstraintMinutes != nil && *timeConstraintMinutes > 0 && opt.DurationMinutes != nil {
			// Include if option duration fits within constraint
			if dur, ok := opt.DurationMinutes.(int64); ok {
				if dur > *timeConstraintMinutes {
					continue // Skip if duration exceeds constraint
				}
			}
		}

		// Tag filter
		//nolint:nestif
		if len(selectedTags) > 0 {
			// Fetch tags for this option
			optionTags, err := h.fetchTagsForOption(ctx, opt.ID, userID)
			if err != nil {
				h.Logger.Warn("Failed to fetch tags for filtering", "option_id", opt.ID, "error", err)
				continue
			}

			// Include if: no tags OR has at least one matching tag
			if len(optionTags) > 0 {
				hasMatch := false
				for _, selectedTag := range selectedTags {
					if slices.Contains(optionTags, selectedTag) {
						hasMatch = true
						break
					}
					if hasMatch {
						break
					}
				}
				if !hasMatch {
					continue // Skip if no matching tags
				}
			}
			// home.Options with no tags always pass (len(optionTags) == 0)
		}

		eligibleOptions = append(eligibleOptions, opt)
	}

	// If no eligible options after filtering, return indicator
	if len(eligibleOptions) == 0 {
		return home.Option{}, true, nil // true indicates "no options available" due to constraint
	}

	// Weighted selection algorithm
	var totalWeight int64
	for _, opt := range eligibleOptions {
		if opt.Weight.Valid {
			totalWeight += opt.Weight.Int64
		} else {
			totalWeight += 1 // Default weight
		}
	}

	if totalWeight == 0 {
		return h.dbOptionToAppOption(ctx, eligibleOptions[0], userID), false, nil
	}

	//nolint:gosec
	r := rand.Int64N(totalWeight)
	var currentWeight int64
	for _, opt := range eligibleOptions {
		weight := int64(1)
		if opt.Weight.Valid {
			weight = opt.Weight.Int64
		}
		currentWeight += weight
		if r < currentWeight {
			return h.dbOptionToAppOption(ctx, opt, userID), false, nil
		}
	}

	return h.dbOptionToAppOption(ctx, eligibleOptions[0], userID), false, nil
}

// AddOption handles adding a new option
func (h *Handler) AddOption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	text := r.FormValue("text")
	durationStr := r.FormValue("duration")

	if strings.TrimSpace(text) == "" {
		h.Logger.Error("Empty option text", "text", text)
		http.Error(w, "Option text cannot be empty", http.StatusBadRequest)
		return
	}

	// Parse duration (optional field)
	var duration *int64
	if durationStr != "" {
		dur, err := strconv.ParseInt(durationStr, 10, 64)
		if err != nil || dur < 0 || dur > 1440 {
			h.Logger.Error("Invalid duration", "duration", durationStr, "error", err)
			http.Error(w, "Invalid duration (must be 0-1440 minutes)", http.StatusBadRequest)
			return
		}
		duration = &dur
	}

	// Create option in database
	var durationParam any
	if duration != nil {
		durationParam = *duration
	}

	createParams := queries.CreateOptionParams{
		Name:            text,
		DurationMinutes: durationParam,
		Weight:          sql.NullInt64{Int64: 1, Valid: true}, // Default weight
		UserID:          userID,
	}

	createdOption, err := h.Database.Queries().CreateOption(ctx, createParams)
	if err != nil {
		h.Logger.Error("Failed to create option", "error", err)
		http.Error(w, "Failed to create option", http.StatusInternalServerError)
		return
	}

	// Parse and set tags
	tagsStr := r.FormValue("tags")
	tags := parseTagsFromForm(tagsStr)
	if len(tags) > 0 {
		if err := h.setTagsForOption(ctx, createdOption.ID, userID, tags); err != nil {
			h.Logger.Error("Failed to set tags", "error", err)
			// Continue - tags are optional
		}
	}

	h.Logger.Info("Option created", "text", text, "duration", duration, "tags", tags)

	// Return updated list to refresh display
	options, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get options", "error", err)
		http.Error(w, "Failed to get options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(options))
	for i, opt := range options {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// Home handles the home page
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user email from context (set by UserContextMiddleware)
	userEmail := utils.GetUserEmail(r)

	// Fetch all available tags for the filter (only for authenticated users)
	var allTags []queries.Tag
	userID, ok := utils.GetUserID(r)
	if ok {
		tags, err := h.Database.Queries().GetAllTags(ctx, userID)
		if err != nil {
			h.Logger.Warn("Failed to fetch tags for filter", "error", err)
			allTags = []queries.Tag{} // Empty slice on error
		} else {
			allTags = tags
		}
	} else {
		allTags = []queries.Tag{} // No tags for anonymous users
	}

	h.html(ctx, w, http.StatusOK, core.HTML("Example Site", home.Page(allTags, userEmail), userEmail))
}

// RandomPicker handles the random activity picker request
func (h *Handler) RandomPicker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	// Parse time constraint from form
	hoursStr := r.FormValue("hours")
	minutesStr := r.FormValue("minutes")

	var timeConstraintMinutes *int64
	if hoursStr != "" || minutesStr != "" {
		hours, _ := strconv.ParseInt(hoursStr, 10, 64)
		minutes, _ := strconv.ParseInt(minutesStr, 10, 64)
		totalMinutes := (hours * 60) + minutes

		// Only apply constraint if total > 0
		if totalMinutes > 0 {
			timeConstraintMinutes = &totalMinutes
		}
	}

	// Parse tag filter from form
	if err := r.ParseForm(); err != nil {
		h.Logger.Error("Failed to parse form", "error", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	selectedTags := r.Form["tags[]"] // Get array of selected tags

	// Add delay to let spinner show
	time.Sleep(800 * time.Millisecond)

	selected, noOptionsAvailable, err := h.selectRandomOption(r.Context(), userID, timeConstraintMinutes, selectedTags)
	if err != nil {
		h.Logger.Error("Failed to select random option", "error", err)
		http.Error(w, "Failed to select option", http.StatusInternalServerError)
		return
	}

	// Handle case where no options match the filters
	if noOptionsAvailable {
		// Use a default time value for the message if no constraint was set
		constraintMinutes := int64(0)
		if timeConstraintMinutes != nil {
			constraintMinutes = *timeConstraintMinutes
		}
		h.html(r.Context(), w, http.StatusOK, home.NoOptionsAvailable(constraintMinutes))
		return
	}

	// Calculate probability
	options, err := h.Database.Queries().GetOptions(r.Context(), userID)
	if err != nil {
		h.Logger.Error("Failed to get options for probability", "error", err)
		http.Error(w, "Failed to calculate probability", http.StatusInternalServerError)
		return
	}

	var totalWeight int64
	var optionWeight int64
	selectedID, _ := stringToInt64(selected.ID)

	for _, opt := range options {
		weight := int64(1)
		if opt.Weight.Valid {
			weight = opt.Weight.Int64
		}
		totalWeight += weight
		if opt.ID == selectedID {
			optionWeight = weight
		}
	}

	probability := float64(optionWeight) / float64(totalWeight)
	result := home.Result(selected.Text, probability, selected.Duration)
	h.html(r.Context(), w, http.StatusOK, result)
}

// GetOptions handles fetching all options for management
func (h *Handler) GetOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	options, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get options", "error", err)
		http.Error(w, "Failed to get options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(options))
	for i, opt := range options {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.ManageModal(appOptions, totalWeight))
}

// UpdateOption handles updating an existing option
func (h *Handler) UpdateOption(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	text := r.FormValue("text")
	idStr := r.FormValue("id")

	if strings.TrimSpace(text) == "" {
		http.Error(w, "Option text cannot be empty", http.StatusBadRequest)
		return
	}

	id, err := stringToInt64(idStr)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Get current option to preserve other fields
	dbOpt, err := h.Database.Queries().GetOption(ctx, queries.GetOptionParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Option not found", http.StatusNotFound)
		return
	}

	updateParams := queries.UpdateOptionParams{
		Name:            text,
		Bio:             dbOpt.Bio,
		DurationMinutes: dbOpt.DurationMinutes,
		Weight:          dbOpt.Weight,
		ID:              id,
		UserID:          userID,
	}

	err = h.Database.Queries().UpdateOption(ctx, updateParams)
	if err != nil {
		h.Logger.Error("Failed to update option", "error", err)
		http.Error(w, "Failed to update option", http.StatusInternalServerError)
		return
	}

	// Return updated options list to refresh all probabilities
	updatedOptions, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get updated options", "error", err)
		http.Error(w, "Failed to refresh options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(updatedOptions))
	for i, opt := range updatedOptions {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// UpdateDuration handles updating option duration
func (h *Handler) UpdateDuration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	id := r.FormValue("id")
	durationStr := r.FormValue("duration")

	if id == "" {
		http.Error(w, "Option ID required", http.StatusBadRequest)
		return
	}

	intID, err := stringToInt64(id)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Parse duration
	var duration any
	if durationStr != "" {
		dur, err := strconv.ParseInt(durationStr, 10, 64)
		if err != nil || dur < 0 || dur > 1440 {
			h.Logger.Error("Invalid duration", "duration", durationStr, "error", err)
			http.Error(w, "Invalid duration (must be 0-1440 minutes)", http.StatusBadRequest)
			return
		}
		duration = dur
	}

	updateParams := queries.UpdateDurationParams{
		DurationMinutes: duration,
		ID:              intID,
		UserID:          userID,
	}

	err = h.Database.Queries().UpdateDuration(ctx, updateParams)
	if err != nil {
		h.Logger.Error("Failed to update duration", "error", err)
		http.Error(w, "Failed to update duration", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("Duration updated", "id", id, "duration", duration)
	w.Header().Set("HX-Trigger", `{"success": "Duration updated successfully"}`)

	// Return updated options list to refresh all probabilities
	updatedOptions, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get updated options", "error", err)
		http.Error(w, "Failed to refresh options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(updatedOptions))
	for i, opt := range updatedOptions {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// IncreaseWeight handles weight increase
func (h *Handler) IncreaseWeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	intID, err := stringToInt64(id)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Get current option
	dbOpt, err := h.Database.Queries().GetOption(ctx, queries.GetOptionParams{
		ID:     intID,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Option not found", http.StatusNotFound)
		return
	}

	// Calculate new weight
	currentWeight := int64(1)
	if dbOpt.Weight.Valid {
		currentWeight = dbOpt.Weight.Int64
	}
	newWeight := min(currentWeight+1, 10)

	updateParams := queries.UpdateWeightParams{
		Weight: sql.NullInt64{Int64: newWeight, Valid: true},
		ID:     intID,
		UserID: userID,
	}

	err = h.Database.Queries().UpdateWeight(ctx, updateParams)
	if err != nil {
		h.Logger.Error("Failed to increase weight", "error", err)
		http.Error(w, "Failed to increase weight", http.StatusInternalServerError)
		return
	}

	// Return updated options list to refresh all probabilities
	updatedOptions, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get updated options", "error", err)
		http.Error(w, "Failed to refresh options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(updatedOptions))
	for i, opt := range updatedOptions {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// DecreaseWeight handles weight decrease
func (h *Handler) DecreaseWeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	intID, err := stringToInt64(id)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Get current option
	dbOpt, err := h.Database.Queries().GetOption(ctx, queries.GetOptionParams{
		ID:     intID,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Option not found", http.StatusNotFound)
		return
	}

	// Calculate new weight
	currentWeight := int64(1)
	if dbOpt.Weight.Valid {
		currentWeight = dbOpt.Weight.Int64
	}
	newWeight := max(currentWeight-1, 1)

	updateParams := queries.UpdateWeightParams{
		Weight: sql.NullInt64{Int64: newWeight, Valid: true},
		ID:     intID,
		UserID: userID,
	}

	err = h.Database.Queries().UpdateWeight(ctx, updateParams)
	if err != nil {
		h.Logger.Error("Failed to decrease weight", "error", err)
		http.Error(w, "Failed to decrease weight", http.StatusInternalServerError)
		return
	}

	// Return updated options list to refresh all probabilities
	updatedOptions, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get updated options", "error", err)
		http.Error(w, "Failed to refresh options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(updatedOptions))
	for i, opt := range updatedOptions {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// DeleteOption handles option deletion
func (h *Handler) DeleteOption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	intID, err := stringToInt64(id)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	err = h.Database.Queries().DeleteOption(ctx, queries.DeleteOptionParams{
		ID:     intID,
		UserID: userID,
	})
	if err != nil {
		h.Logger.Error("Failed to delete option", "error", err)
		http.Error(w, "Failed to delete option", http.StatusInternalServerError)
		return
	}

	// Return updated options list to refresh all probabilities
	updatedOptions, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get updated options", "error", err)
		http.Error(w, "Failed to refresh options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(updatedOptions))
	for i, opt := range updatedOptions {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// ExpandOption handles showing the expanded edit form
func (h *Handler) ExpandOption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	// Extract ID from path: /expand-option/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}
	id := parts[2]

	ctx := r.Context()
	intID, err := stringToInt64(id)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Get the specific option
	dbOpt, err := h.Database.Queries().GetOption(ctx, queries.GetOptionParams{
		ID:     intID,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Option not found", http.StatusNotFound)
		return
	}

	// Get all options to calculate total weight
	options, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get options", "error", err)
		http.Error(w, "Failed to get options", http.StatusInternalServerError)
		return
	}

	var totalWeight int64
	for _, opt := range options {
		weight := int64(1)
		if opt.Weight.Valid {
			weight = opt.Weight.Int64
		}
		totalWeight += weight
	}

	appOption := h.dbOptionToAppOption(ctx, dbOpt, userID)
	h.html(ctx, w, http.StatusOK, home.ExpandedOptionRow(appOption, totalWeight))
}

// CollapseOption handles collapsing the expanded edit form
func (h *Handler) CollapseOption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	// Extract ID from path: /collapse-option/{id}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid path format", http.StatusBadRequest)
		return
	}
	id := parts[2]

	ctx := r.Context()
	intID, err := stringToInt64(id)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Get the specific option
	dbOpt, err := h.Database.Queries().GetOption(ctx, queries.GetOptionParams{
		ID:     intID,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Option not found", http.StatusNotFound)
		return
	}

	// Get all options to calculate total weight
	options, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get options", "error", err)
		http.Error(w, "Failed to get options", http.StatusInternalServerError)
		return
	}

	var totalWeight int64
	for _, opt := range options {
		weight := int64(1)
		if opt.Weight.Valid {
			weight = opt.Weight.Int64
		}
		totalWeight += weight
	}

	appOption := h.dbOptionToAppOption(ctx, dbOpt, userID)
	h.html(ctx, w, http.StatusOK, home.OptionRow(appOption, totalWeight))
}

// UpdateOptionDetails handles updating both duration and weight
func (h *Handler) UpdateOptionDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := utils.RequireAuth(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	// Parse form values
	idStr := r.FormValue("id")
	textStr := r.FormValue("text")
	hoursStr := r.FormValue("hours")
	minutesStr := r.FormValue("minutes")
	weightStr := r.FormValue("weight")

	id, err := stringToInt64(idStr)
	if err != nil {
		http.Error(w, "Invalid option ID", http.StatusBadRequest)
		return
	}

	// Validate name
	if strings.TrimSpace(textStr) == "" {
		http.Error(w, "Option name cannot be empty", http.StatusBadRequest)
		return
	}

	// Parse and clamp hours (0-24)
	hours, err := strconv.ParseInt(hoursStr, 10, 64)
	if err != nil {
		hours = 0
	}
	if hours < 0 {
		hours = 0
	}
	if hours > 24 {
		hours = 24
	}

	// Parse and clamp minutes (0-59)
	minutes, err := strconv.ParseInt(minutesStr, 10, 64)
	if err != nil {
		minutes = 0
	}
	if minutes < 0 {
		minutes = 0
	}
	if minutes > 59 {
		minutes = 59
	}

	// Parse and clamp weight (1-10)
	weight, err := strconv.ParseInt(weightStr, 10, 64)
	if err != nil {
		weight = 1
	}
	if weight < 1 {
		weight = 1
	}
	if weight > 10 {
		weight = 10
	}

	// Calculate total minutes
	var duration any
	totalMinutes := (hours * 60) + minutes
	if totalMinutes > 0 {
		duration = totalMinutes
	} else {
		duration = nil
	}

	// Get current option to preserve other fields
	dbOpt, err := h.Database.Queries().GetOption(ctx, queries.GetOptionParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "Option not found", http.StatusNotFound)
		return
	}

	// Update option with name, duration, and weight
	updateParams := queries.UpdateOptionParams{
		Name:            textStr,
		Bio:             dbOpt.Bio,
		DurationMinutes: duration,
		Weight:          sql.NullInt64{Int64: weight, Valid: true},
		ID:              id,
		UserID:          userID,
	}

	err = h.Database.Queries().UpdateOption(ctx, updateParams)
	if err != nil {
		h.Logger.Error("Failed to update option", "error", err)
		http.Error(w, "Failed to update option", http.StatusInternalServerError)
		return
	}

	// Parse and set tags
	tagsStr := r.FormValue("tags")
	tags := parseTagsFromForm(tagsStr)
	if err := h.setTagsForOption(ctx, id, userID, tags); err != nil {
		h.Logger.Error("Failed to update tags", "error", err)
		http.Error(w, "Failed to update tags", http.StatusInternalServerError)
		return
	}

	h.Logger.Info("Option updated", "id", id, "name", textStr, "duration", totalMinutes, "weight", weight, "tags", tags)

	// Return full options list to refresh all probabilities
	options, err := h.Database.Queries().GetOptions(ctx, userID)
	if err != nil {
		h.Logger.Error("Failed to get options", "error", err)
		http.Error(w, "Failed to refresh options", http.StatusInternalServerError)
		return
	}

	appOptions := make([]home.Option, len(options))
	for i, opt := range options {
		appOptions[i] = h.dbOptionToAppOption(ctx, opt, userID)
	}

	var totalWeight int64
	for _, opt := range appOptions {
		totalWeight += opt.Weight
	}

	h.html(ctx, w, http.StatusOK, home.OptionsListWithWeight(appOptions, totalWeight))
}

// CloseModal handles closing the management modal
func (h *Handler) CloseModal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.html(r.Context(), w, http.StatusOK, home.CloseModal())
}

// extractIDFromPath extracts option ID from URL path
func extractIDFromPath(path string) string {
	// Path format: /api/weight/{action}/{id}, /api/options/{id}, /edit-duration/{id}, /cancel-duration-edit/{id}
	parts := strings.Split(path, "/")

	// Handle /api/weight/{action}/{id} format (5 parts)
	if len(parts) >= 5 && parts[1] == "api" && parts[2] == "weight" {
		return parts[4]
	}

	// Handle /api/options/{id}, /edit-duration/{id}, /cancel-duration-edit/{id} format (4 parts)
	if len(parts) >= 4 {
		return parts[3]
	}

	return ""
}
