package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"web/config"
	"web/templates/components"
	wikipages "web/templates/wiki-pages"
	"web/utils"

	"github.com/gin-gonic/gin"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// GetPageHistory renders the split-view revision history page
func GetPageHistory(c *gin.Context) {
	id := c.Param("id")
	revId := c.Param("revId")

	// Get current page data
	page, err := fetchPageData(id)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// Get revisions - fetch more to ensure we can find previous revision for older entries
	revisions, err := fetchRevisions(id, 0, 20)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// Determine which revision to show
	var currentRevision utils.RevisionDetail
	var revisionNumber int

	if revId != "" {
		// Fetch specific revision
		currentRevision, err = fetchRevision(id, revId)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		// Find the revision number (position in list, with oldest as #1)
		for i, rev := range revisions {
			if rev.UUID != nil && *rev.UUID == currentRevision.UUID {
				revisionNumber = len(revisions) - i
				break
			}
		}
	} else {
		// Show the most recent revision (first in the list since sorted newest first)
		if len(revisions) > 0 {
			// Fetch the full revision content since the list endpoint may not include it
			currentRevision, err = fetchRevision(id, revisions[0].UUID.String())
			if err != nil {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
			revisionNumber = len(revisions)
		}
	}

	// Get previous revision for diff highlighting
	var previousRevision *utils.RevisionDetail
	for i, rev := range revisions {
		if rev.UUID != nil && *rev.UUID == currentRevision.UUID && i < len(revisions)-1 {
			// Fetch the full previous revision content since the list doesn't include it
			prevRevId := revisions[i+1].UUID.String()
			prevRev, err := fetchRevision(id, prevRevId)
			if err == nil {
				previousRevision = &prevRev
			}
			break
		}
	}

	// Highlight changes and convert to HTML
	highlightedContent, hasChanges := highlightChanges(currentRevision.Content, previousRevision)

	// Convert new types to deprecated Revision type for template compatibility
	revisionsForTemplate := utils.RevisionListToRevisions(revisions)
	currentRevisionForTemplate := currentRevision.ToRevision()

	// Check if HTMX request (for partial content update)
	if c.GetHeader("HX-Request") == "true" {
		// Return article content AND updated timeline selection
		// Article replaces #article-content via hx-target
		articleContent := wikipages.WikiHistoryArticle(page, currentRevisionForTemplate, highlightedContent, revisionNumber, hasChanges)
		articleContent.Render(context.Background(), c.Writer)

		// Timeline updates selection via hx-swap-oob
		timelineContent := wikipages.WikiHistoryTimeline(revisionsForTemplate, currentRevision.UUID.String(), len(revisions))
		timelineContent.Render(context.Background(), c.Writer)
		return
	}

	// Full page render
	historyContent := wikipages.WikiHistoryContent(page, revisionsForTemplate, currentRevisionForTemplate, highlightedContent, revisionNumber, hasChanges)
	component := components.Page(page.Name+" - Revision History", historyContent)
	component.Render(context.Background(), c.Writer)
}

// GetTimelinePartial returns more timeline items for infinite scroll
func GetTimelinePartial(c *gin.Context) {
	id := c.Param("id")

	indexStr := c.Query("index")
	index := 0
	if indexStr != "" {
		index, _ = strconv.Atoi(indexStr)
	}

	revisions, err := fetchRevisions(id, index, 20)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// Get total count for numbering
	totalRevisions, _ := fetchRevisions(id, 0, 1000)
	totalCount := len(totalRevisions)

	// Convert new types to deprecated Revision type for template compatibility
	revisionsForTemplate := utils.RevisionListToRevisions(revisions)

	timelineItems := wikipages.WikiHistoryTimelineItems(revisionsForTemplate, totalCount, index)
	timelineItems.Render(context.Background(), c.Writer)
}

// fetchPageData gets page data from API
func fetchPageData(id string) (utils.Page, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pages/%s", config.WikiURL, id))
	if err != nil {
		return utils.Page{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.Page{}, err
	}

	var page utils.Page
	err = json.Unmarshal(body, &page)
	if err != nil {
		return utils.Page{}, err
	}

	return page, nil
}

// fetchRevisions gets revision list from API
func fetchRevisions(id string, index, count int) ([]utils.RevisionList, error) {
	url := fmt.Sprintf("%s/pages/%s/revisions?index=%d&count=%d", config.WikiURL, id, index, count)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var revisions []utils.RevisionList
	err = json.Unmarshal(body, &revisions)
	if err != nil {
		return nil, err
	}

	return revisions, nil
}

// fetchRevision gets a specific revision from API
func fetchRevision(id, revId string) (utils.RevisionDetail, error) {
	url := fmt.Sprintf("%s/pages/%s/revisions/%s", config.WikiURL, id, revId)
	resp, err := http.Get(url)
	if err != nil {
		return utils.RevisionDetail{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.RevisionDetail{}, err
	}

	var revision utils.RevisionDetail
	err = json.Unmarshal(body, &revision)
	if err != nil {
		return utils.RevisionDetail{}, err
	}

	return revision, nil
}

// highlightChanges compares content and shows deleted text with strikethrough
// Returns the HTML content with deletions shown and a boolean indicating if there are changes
func highlightChanges(currentContent string, previousRevision *utils.RevisionDetail) (string, bool) {
	// Convert current content to HTML
	currentHTML, err := utils.ToHTML(currentContent)
	if err != nil {
		currentHTML = currentContent
	}

	if previousRevision == nil {
		// No previous revision, return HTML as-is with no changes
		return currentHTML, false
	}

	// Convert previous content to HTML
	previousHTML, err := utils.ToHTML(previousRevision.Content)
	if err != nil {
		previousHTML = previousRevision.Content
	}

	// If HTML outputs are identical, no changes to highlight
	if currentHTML == previousHTML {
		return currentHTML, false
	}

	// Use diffmatchpatch to find differences
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(previousHTML, currentHTML, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	// Check if there are actual changes
	hasChanges := false
	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffInsert || diff.Type == diffmatchpatch.DiffDelete {
			hasChanges = true
			break
		}
	}

	if !hasChanges {
		return currentHTML, false
	}

	// Build result showing current content with deletions as strikethrough
	var result strings.Builder
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			// Added content - wrap with appropriate element based on content type
			text := diff.Text
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				result.WriteString(text)
			} else if containsBlockLevelTags(trimmed) {
				// Block-level content - wrap in div
				result.WriteString(`<div class="revision-added-block">`)
				result.WriteString(text)
				result.WriteString(`</div>`)
			} else {
				// Inline content - wrap in span
				result.WriteString(`<span class="revision-added">`)
				result.WriteString(text)
				result.WriteString(`</span>`)
			}
		case diffmatchpatch.DiffDelete:
			// Deleted content - wrap with appropriate element based on content type
			text := diff.Text
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				// Skip whitespace-only deletions
			} else if containsBlockLevelTags(trimmed) {
				// Block-level content - wrap in div
				result.WriteString(`<div class="revision-deleted-block">`)
				result.WriteString(text)
				result.WriteString(`</div>`)
			} else {
				// Inline content - wrap in span with strikethrough
				result.WriteString(`<span class="revision-deleted">`)
				result.WriteString(text)
				result.WriteString(`</span>`)
			}
		case diffmatchpatch.DiffEqual:
			// Unchanged content - show as-is
			result.WriteString(diff.Text)
		}
	}

	return result.String(), true
}

// containsBlockLevelTags checks if the text contains block-level HTML elements
// that shouldn't be wrapped in inline span elements
func containsBlockLevelTags(text string) bool {
	// List of block-level HTML tags that shouldn't be wrapped in inline spans
	blockLevelTags := []string{
		"<p", "</p",
		"<div", "</div",
		"<h1", "</h1",
		"<h2", "</h2",
		"<h3", "</h3",
		"<h4", "</h4",
		"<h5", "</h5",
		"<h6", "</h6",
		"<ul", "</ul",
		"<ol", "</ol",
		"<li", "</li",
		"<table", "</table",
		"<thead", "</thead",
		"<tbody", "</tbody",
		"<tr", "</tr",
		"<blockquote", "</blockquote",
		"<pre", "</pre",
		"<hr",
	}

	lowerText := strings.ToLower(text)
	for _, tag := range blockLevelTags {
		if strings.Contains(lowerText, tag) {
			return true
		}
	}
	return false
}
