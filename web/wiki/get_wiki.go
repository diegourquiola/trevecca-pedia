package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"web/config"
	"web/templates/components"
	wikipages "web/templates/wiki-pages"
	"web/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func GetPage(c *gin.Context) {
	id := c.Param("id")
	resp, err := http.Get(fmt.Sprintf("%s/pages/%s", config.WikiURL, id))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Couldn't read http request: %w\n", err))
	}

	var page utils.Page
	err = json.Unmarshal(body, &page)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("Couldn't parse json from API layer: %w\n", err))
	}

	entryContent := wikipages.WikiEntryContent(page)
	component := components.Page(page.Name, entryContent)
	component.Render(context.Background(), c.Writer)

}

func GetHome(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	pages, err := getPages()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Couldn't fetch page info: %w\n", err))
	}
	if len(pages) == 0 {
		pages = append(pages, utils.PageInfoPrev{UUID: uuid.UUID{}, Slug: "", Name: "Not Found", LastEditTime: time.Time{}, ArchiveDate: time.Time{}, Preview: "No pages found"})
	}
	homeComp := components.HomeContent(pages)
	page := components.Page("TreveccaPedia", homeComp)
	page.Render(context.Background(), c.Writer)
}

func getPages() ([]utils.PageInfoPrev, error) {
	resp, err := http.Get(fmt.Sprintf("%s/pages", config.WikiURL))
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
