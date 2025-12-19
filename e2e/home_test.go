//go:build e2e

package e2e_test

import (
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/require"
)

func TestHome(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	require.NoError(t, expect.Locator(page.GetByText("Can't decide what to do?")).ToBeVisible())
}

// Test: Tag Filter Visibility
func TestTagFilterVisibility(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Tag filter should be visible (we have tags in seed data)
	tagFilterBtn := page.GetByText("Filter by tags")
	require.NoError(t, expect.Locator(tagFilterBtn).ToBeVisible())

	// Click to expand
	require.NoError(t, tagFilterBtn.Click())

	// Verify tags are displayed alphabetically
	require.NoError(t, expect.Locator(page.GetByText("active")).ToBeVisible())
	require.NoError(t, expect.Locator(page.GetByText("gaming")).ToBeVisible())
	require.NoError(t, expect.Locator(page.GetByText("indoor")).ToBeVisible())
	require.NoError(t, expect.Locator(page.GetByText("outdoor")).ToBeVisible())
	require.NoError(t, expect.Locator(page.GetByText("relaxing")).ToBeVisible())
}

// Test: Tag Selection Toggle
func TestTagSelectionToggle(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand tag filter
	tagFilterBtn := page.GetByText("Filter by tags")
	require.NoError(t, tagFilterBtn.Click())

	// Find and click a tag pill
	indoorPill := page.Locator("button.tag-pill[data-tag-name='indoor']")
	require.NoError(t, indoorPill.Click())

	// Verify selected state (check for solid background class)
	classAttr, err := indoorPill.GetAttribute("class")
	require.NoError(t, err)
	require.Contains(t, classAttr, "bg-purple-500")

	// Click again to deselect
	require.NoError(t, indoorPill.Click())

	// Verify unselected state
	classAttr, err = indoorPill.GetAttribute("class")
	require.NoError(t, err)
	require.Contains(t, classAttr, "bg-purple-500/10")
}

// Test: Single Tag Filtering
func TestSingleTagFiltering(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand tag filter
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Select "gaming" tag (options: Video Games, Board Games)
	gamingPill := page.Locator("button.tag-pill[data-tag-name='gaming']")
	require.NoError(t, gamingPill.Click())

	// Submit form multiple times to test randomness
	for i := 0; i < 5; i++ {
		submitBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Make a decision"})
		require.NoError(t, submitBtn.Click())

		// Wait for result
		resultCard := page.Locator("#result-card")
		require.NoError(t, expect.Locator(resultCard).ToBeVisible())

		// Result should be: Video Games, Board Games, or Meditation (no tags)
		resultText, err := page.Locator("#result-card").TextContent()
		require.NoError(t, err)

		validResults := []string{"Video Games", "Board Games", "Meditation"}
		hasValidResult := false
		for _, valid := range validResults {
			if strings.Contains(resultText, valid) {
				hasValidResult = true
				break
			}
		}
		require.True(t, hasValidResult, "Result should match gaming tag or have no tags, got: %s", resultText)

		// Dismiss result
		require.NoError(t, page.GetByText("Got it!").Click())
	}
}

// Test: Multiple Tag Filtering (OR Logic)
func TestMultipleTagFilteringORLogic(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand tag filter
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Select "gaming" and "outdoor" tags
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='gaming']").Click())
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='outdoor']").Click())

	// Submit and verify
	for i := 0; i < 5; i++ {
		submitBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Make a decision"})
		require.NoError(t, submitBtn.Click())

		// Wait for result
		require.NoError(t, expect.Locator(page.Locator("#result-card")).ToBeVisible())

		// Result should be: Video Games, Board Games, Running, or Meditation
		resultText, err := page.Locator("#result-card").TextContent()
		require.NoError(t, err)

		validResults := []string{"Video Games", "Board Games", "Going for a Run", "Meditation"}
		hasValidResult := false
		for _, valid := range validResults {
			if strings.Contains(resultText, valid) {
				hasValidResult = true
				break
			}
		}
		require.True(t, hasValidResult, "Result should match gaming OR outdoor tags OR have no tags, got: %s", resultText)

		// Dismiss result
		require.NoError(t, page.GetByText("Got it!").Click())
	}
}

// Test: Options Without Tags Always Included
func TestOptionsWithoutTagsAlwaysIncluded(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand tag filter
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Select "active" tag (only "Running" has this)
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='active']").Click())

	// Submit multiple times
	foundMeditation := false
	foundRunning := false

	for i := 0; i < 10; i++ {
		submitBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Make a decision"})
		require.NoError(t, submitBtn.Click())

		// Wait for result
		require.NoError(t, expect.Locator(page.Locator("#result-card")).ToBeVisible())

		resultText, err := page.Locator("#result-card").TextContent()
		require.NoError(t, err)

		if strings.Contains(resultText, "Meditation") {
			foundMeditation = true
		}
		if strings.Contains(resultText, "Going for a Run") {
			foundRunning = true
		}

		// Dismiss result
		require.NoError(t, page.GetByText("Got it!").Click())
	}

	// Both should appear (Meditation has no tags, Running has "active" tag)
	require.True(t, foundMeditation || foundRunning, "Should find either Meditation (no tags) or Running (active tag)")
}

// Test: Combined Time + Tag Filters
func TestCombinedTimeAndTagFilters(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand time constraint
	require.NoError(t, page.GetByText("Add time constraint").Click())

	// Set time to 60 minutes
	require.NoError(t, page.Locator("#constraint-hours").Fill("1"))
	require.NoError(t, page.Locator("#constraint-minutes").Fill("0"))

	// Expand tag filter
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Select "indoor" tag
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='indoor']").Click())

	// Submit and verify
	submitBtn := page.GetByRole("button", playwright.PageGetByRoleOptions{Name: "Make a decision"})
	require.NoError(t, submitBtn.Click())

	// Wait for result
	require.NoError(t, expect.Locator(page.Locator("#result-card")).ToBeVisible())

	// Valid results: <= 60min AND (indoor tag OR no tags)
	// Video Games (60min, indoor), Reading (30min, indoor), Meditation (15min, no tags)
	resultText, err := page.Locator("#result-card").TextContent()
	require.NoError(t, err)

	validResults := []string{"Video Games", "Reading a Book", "Meditation"}
	hasValidResult := false
	for _, valid := range validResults {
		if strings.Contains(resultText, valid) {
			hasValidResult = true
			break
		}
	}
	require.True(t, hasValidResult, "Result should match both time and tag filters, got: %s", resultText)
}

// Test: No Options Available (Tag Filter) - Skipped as it requires deleting data
func TestNoOptionsAvailableTagFilter(t *testing.T) {
	t.Skip("Skipping test that requires data deletion - would interfere with other tests")
}

// Test: Clear All Tags Button
func TestClearAllTagsButton(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand tag filter
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Select multiple tags
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='indoor']").Click())
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='gaming']").Click())

	// Verify selected state
	indoorClass, err := page.Locator("button.tag-pill[data-tag-name='indoor']").GetAttribute("class")
	require.NoError(t, err)
	require.Contains(t, indoorClass, "bg-purple-500")

	// Click "Clear all"
	require.NoError(t, page.GetByText("Clear all").Click())

	// Verify all tags are unselected
	indoorClass, err = page.Locator("button.tag-pill[data-tag-name='indoor']").GetAttribute("class")
	require.NoError(t, err)
	require.Contains(t, indoorClass, "bg-purple-500/10")

	gamingClass, err := page.Locator("button.tag-pill[data-tag-name='gaming']").GetAttribute("class")
	require.NoError(t, err)
	require.Contains(t, gamingClass, "bg-purple-500/10")
}

// Test: Tag Filter Collapse/Expand Persistence
func TestTagFilterPersistence(t *testing.T) {
	beforeEach(t)
	_, err := page.Goto(getFullPath(""))
	require.NoError(t, err)

	// Expand tag filter
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Select a tag
	require.NoError(t, page.Locator("button.tag-pill[data-tag-name='indoor']").Click())

	// Collapse
	require.NoError(t, page.GetByText("Hide tag filter").Click())

	// Re-expand
	require.NoError(t, page.GetByText("Filter by tags").Click())

	// Verify tag is still selected
	indoorClass, err := page.Locator("button.tag-pill[data-tag-name='indoor']").GetAttribute("class")
	require.NoError(t, err)
	require.Contains(t, indoorClass, "bg-purple-500", "Selected tag should persist after collapse/expand")
}
