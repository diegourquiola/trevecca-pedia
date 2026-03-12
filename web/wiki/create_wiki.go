package wiki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"web/config"
	"web/templates/components"
	wikipages "web/templates/wiki-pages"

	"github.com/gin-gonic/gin"
)

// GetCreatePage renders the "Create New Page" form.
func GetCreatePage(c *gin.Context) {
	categories, _ := getCategories()
	createContent := wikipages.WikiCreateContent("", "", "", "# Title\n\nContent here...", categories, []string{})
	component := components.Page("Create New Page", createContent)
	component.Render(context.Background(), c.Writer)
}

// PostCreatePage handles the form submission for creating a new page.
func PostCreatePage(c *gin.Context) {
	name := c.PostForm("name")
	slug := c.PostForm("slug")
	content := c.PostForm("content")

	// Fetch categories for re-rendering on error
	categories, _ := getCategories()

	// Render helper: re-renders the form preserving user input + showing error.
	renderErr := func(errMsg string) {
		createContent := wikipages.WikiCreateContent(errMsg, name, slug, content, categories, []string{})
		component := components.Page("Create New Page", createContent)
		component.Render(context.Background(), c.Writer)
	}

	// Step 1 — resolve the authenticated user's email
	authorEmail, authErr := resolveAuthorEmail(c)
	if authErr != nil {
		renderErr(authErr.Error())
		return
	}

	// Step 2 — validate required fields
	if name == "" {
		renderErr("Page title is required.")
		return
	}
	if slug == "" {
		renderErr("URL slug is required.")
		return
	}
	if content == "" {
		renderErr("Content cannot be empty.")
		return
	}

	// Step 3 — build multipart form for the API layer
	//
	// API: POST /v1/wiki/pages/new
	// Fields:
	//   slug     — the page slug (lowercase kebab-case)
	//   name     — the page display title
	//   author   — user's email
	//   new_page — the markdown content as a file upload
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	writer.WriteField("slug", slug)
	writer.WriteField("name", name)
	writer.WriteField("author", authorEmail)

	filePart, err := writer.CreateFormFile("new_page", "content.md")
	if err != nil {
		renderErr("Internal error preparing page content.")
		return
	}
	filePart.Write([]byte(content))
	writer.Close()

	// Step 4 — send to API layer with Bearer auth
	createURL := fmt.Sprintf("%s/pages/new", config.WikiURL)

	token, _ := c.Cookie(authCookieName)
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		createURL,
		&body,
	)
	if err != nil {
		renderErr("Internal error creating request.")
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := wikiClient.Do(req)
	if err != nil {
		renderErr("Unable to create page. The wiki service is unreachable.")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to extract an error message from the API response
		respBody, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("Unable to create page. (status %d)", resp.StatusCode)
		if len(respBody) > 0 {
			errMsg = fmt.Sprintf("Unable to create page: %s", string(respBody))
		}
		renderErr(errMsg)
		return
	}

	// Step 5 — set categories for the new page (best effort - don't fail if this errors)
	selectedCategories := c.PostFormArray("categories")
	if len(selectedCategories) > 0 {
		token, _ := c.Cookie(authCookieName)
		setPageCategories(c.Request.Context(), token, slug, selectedCategories)
	}

	// Step 6 — success, redirect to the new page
	c.Redirect(http.StatusFound, fmt.Sprintf("/pages/%s?saved=true", slug))
}

// setPageCategories associates categories with a page. Called best-effort after page creation.
func setPageCategories(ctx context.Context, token string, slug string, categorySlugs []string) {
	categoriesURL := fmt.Sprintf("%s/pages/%s/categories", config.WikiURL, slug)

	body, err := json.Marshal(categorySlugs)
	if err != nil {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, categoriesURL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := wikiClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
