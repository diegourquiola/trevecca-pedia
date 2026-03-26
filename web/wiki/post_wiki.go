package wiki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
	"web/config"
	"web/templates/components"
	wikipages "web/templates/wiki-pages"
	"web/utils"

	"github.com/gin-gonic/gin"
)

const authCookieName = "auth_token"

// wikiClient is used for upstream calls to the API layer.
var wikiClient = &http.Client{Timeout: 15 * time.Second}

// resolveAuthorEmail reads the auth_token cookie and calls the auth service's
// /me endpoint to resolve the authenticated user's email address.
// Returns the email string or an error message suitable for display.
func resolveAuthorEmail(c *gin.Context) (string, error) {
	token, err := c.Cookie(authCookieName)
	if err != nil || token == "" {
		return "", fmt.Errorf("You must be logged in to edit pages.")
	}

	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodGet,
		config.AuthURL+"/me",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("Internal error preparing auth request.")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := wikiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Auth service is unreachable.")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Your session has expired. Please log in again.")
	}

	var user struct {
		Email string `json:"email"`
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read auth response.")
	}
	if err := json.Unmarshal(body, &user); err != nil || user.Email == "" {
		return "", fmt.Errorf("Could not determine your identity. Please log in again.")
	}

	return user.Email, nil
}

func PostEditPage(c *gin.Context) {
	id := c.Param("id")

	// Step 1 — resolve the authenticated user's email for the author field
	authorEmail, authErr := resolveAuthorEmail(c)
	if authErr != nil {
		// We still need the page data to re-render the edit form with an error.
		page, fetchErr := fetchPage(id)
		if fetchErr != nil {
			c.AbortWithError(http.StatusBadGateway, fetchErr)
			return
		}
		editContent := wikipages.WikiEditContent(page, authErr.Error())
		component := components.Page("Editing: "+page.Name, editContent)
		component.Render(context.Background(), c.Writer)
		return
	}

	// Step 2 — read the textarea content
	content := c.PostForm("content")
	if content == "" {
		page, fetchErr := fetchPage(id)
		if fetchErr != nil {
			c.AbortWithError(http.StatusBadGateway, fetchErr)
			return
		}
		editContent := wikipages.WikiEditContent(page, "Content cannot be empty.")
		component := components.Page("Editing: "+page.Name, editContent)
		component.Render(context.Background(), c.Writer)
		return
	}

	// Fetch page info to include slug and name in the request
	page, fetchErr := fetchPage(id)
	if fetchErr != nil {
		c.AbortWithError(http.StatusBadGateway, fetchErr)
		return
	}

	// Step 3 — build multipart form for the API layer
	//
	// API: POST /v1/wiki/pages/:id/revisions
	// Fields:
	//   page_id     — the slug (or uuid) of the page
	//   author      — user's email
	//   slug        — the page slug
	//   name        — the page name
	//   new_content — the markdown content as a file upload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writer.WriteField("page_id", id)
	writer.WriteField("author", authorEmail)
	writer.WriteField("slug", page.Slug)
	writer.WriteField("name", page.Name)

	filePart, err := writer.CreateFormFile("new_content", "content.md")
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to create form file: %w", err))
		return
	}
	filePart.Write([]byte(content))
	writer.Close()

	// Step 4 — send to API layer with Bearer auth
	revisionURL := fmt.Sprintf("%s/pages/%s/revisions", config.WikiURL, id)

	token, _ := c.Cookie(authCookieName)
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		revisionURL,
		&body,
	)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to create request: %w", err))
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := wikiClient.Do(req)
	if err != nil {
		// Network error — re-render form with error
		editContent := wikipages.WikiEditContent(page, "Unable to save changes. The wiki service is unreachable.")
		component := components.Page("Editing: "+page.Name, editContent)
		component.Render(context.Background(), c.Writer)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Unable to save changes. (status %d)", resp.StatusCode)
		editContent := wikipages.WikiEditContent(page, errMsg)
		component := components.Page("Editing: "+page.Name, editContent)
		component.Render(context.Background(), c.Writer)
		return
	}

	// Step 5 — success, redirect back to the page
	c.Redirect(http.StatusFound, fmt.Sprintf("/pages/%s?saved=true", id))
}

// fetchPage fetches a page by slug from the wiki API.
func fetchPage(slug string) (utils.Page, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pages/%s", config.WikiURL, slug))
	if err != nil {
		return utils.Page{}, fmt.Errorf("wiki service unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return utils.Page{}, fmt.Errorf("failed to read response: %w", err)
	}

	var page utils.Page
	if err := json.Unmarshal(body, &page); err != nil {
		return utils.Page{}, fmt.Errorf("failed to parse page data: %w", err)
	}

	return page, nil
}
