package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"web/auth"
	"web/config"
	categorytemplates "web/templates/category"
	"web/templates/components"
	wikipages "web/templates/wiki-pages"
	"web/utils"

	"github.com/gin-gonic/gin"
)

func GetPage(c *gin.Context) {
	id := c.Param("id")

	// Check if user is a moderator
	user, _ := auth.GetUserFromContext(c)
	isModerator := auth.HasRole(user, "moderator")

	// Fetch page and categories in parallel
	pageResp, err := http.Get(fmt.Sprintf("%s/pages/%s", config.WikiURL, id))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	defer pageResp.Body.Close()

	pageBody, err := io.ReadAll(pageResp.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Couldn't read http request: %w\n", err))
		return
	}

	var page utils.Page
	err = json.Unmarshal(pageBody, &page)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Couldn't parse json from API layer: %w\n", err))
		return
	}

	// Fetch categories for this page
	categories, _ := getPageCategories(page.UUID.String())
	page.Categories = categories

	saved := c.Query("saved") == "true"
	entryContent := wikipages.WikiEntryContent(page, saved, isModerator)
	component := components.Page(page.Name, entryContent)
	component.Render(context.Background(), c.Writer)
}

func GetHome(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	categories, err := getCategories()
	if err != nil {
		categories = []utils.Category{}
	}
	deleted := c.Query("deleted") == "true"
	homeComp := components.HomeContent(categories, deleted)
	page := components.Page("TreveccaPedia", homeComp)
	page.Render(context.Background(), c.Writer)
}

func GetCategoryPages(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	categorySlug := c.Query("category")

	categories, err := getCategories()
	if err != nil {
		categories = []utils.Category{}
	}

	pages, err := getPagesByCategory(categorySlug)
	if err != nil {
		pages = []utils.PageInfoPrev{}
	}

	// Resolve category name from slug
	flatCategories := flattenCategories(categories)
	categoryName := "All Categories"
	for _, cat := range flatCategories {
		if cat.FullSlug == categorySlug {
			categoryName = cat.Name
			break
		}
	}

	title := categoryName
	if categorySlug == "" {
		title = "All Pages"
	}

	// Check if this is an htmx request
	if c.GetHeader("HX-Request") == "true" {
		// Return only the content partial (no full page wrapper)
		// The hx-select attribute will extract just #category-main-content
		content := categorytemplates.CategoryContent(categorySlug, categoryName, categories, pages)
		content.Render(context.Background(), c.Writer)
		return
	}

	// Full page render for non-htmx requests
	content := categorytemplates.CategoryContent(categorySlug, categoryName, categories, pages)
	component := components.Page(title, content)
	component.Render(context.Background(), c.Writer)
}

func getPagesByCategory(category string) ([]utils.PageInfoPrev, error) {
	url := fmt.Sprintf("%s/pages", config.WikiURL)
	if category != "" {
		url = fmt.Sprintf("%s/pages?category=%s", config.WikiURL, category)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var pages []utils.PageInfoPrev
	err = json.Unmarshal(body, &pages)
	if err != nil {
		return nil, err
	}

	return pages, nil
}

func getCategories() ([]utils.Category, error) {
	resp, err := http.Get(fmt.Sprintf("%s/categories?tree=true", config.WikiURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var categories []utils.Category
	err = json.Unmarshal(body, &categories)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func flattenCategories(categories []utils.Category) []utils.CategoryFlat {
	var result []utils.CategoryFlat

	var flatten func(cats []utils.Category, depth int)
	flatten = func(cats []utils.Category, depth int) {
		for _, cat := range cats {
			result = append(result, utils.CategoryFlat{
				ID:          cat.ID,
				Slug:        cat.Slug,
				Name:        cat.Name,
				FullSlug:    cat.FullSlug,
				Depth:       depth,
				DisplayName: cat.Name,
			})

			if len(cat.Children) > 0 {
				flatten(cat.Children, depth+1)
			}
		}
	}

	flatten(categories, 0)
	return result
}

func getPageCategories(pageId string) ([]utils.Category, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pages/%s/categories", config.WikiURL, pageId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var categories []utils.Category
	err = json.Unmarshal(body, &categories)
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func GetEditPage(c *gin.Context) {
	id := c.Param("id")
	resp, err := http.Get(fmt.Sprintf("%s/pages/%s", config.WikiURL, id))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("couldn't read http response: %w", err))
		return
	}

	var page utils.Page
	err = json.Unmarshal(body, &page)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("couldn't parse json from API layer: %w", err))
		return
	}

	editContent := wikipages.WikiEditContent(page, "")
	component := components.Page("Editing: "+page.Name, editContent)
	component.Render(context.Background(), c.Writer)
}

// PostPreview handles markdown preview requests from the editor
type PreviewRequest struct {
	Content string `json:"content"`
}

type PreviewResponse struct {
	HTML string `json:"html"`
}

func PostPreview(c *gin.Context) {
	var req PreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	html, err := utils.ToHTML(req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render markdown"})
		return
	}

	c.JSON(http.StatusOK, PreviewResponse{HTML: html})
}
