package search

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"web/config"
	"web/templates/components"
	searchtemplates "web/templates/search"
	"web/utils"

	"github.com/gin-gonic/gin"
)

type SearchResponse struct {
	Total   int      `json:"total"`
	Results []string `json:"results"`
}

func GetSearchPage(c *gin.Context) {
	query := c.Query("q")

	var pages []utils.PageInfoPrev

	if query != "" {
		searchResults, err := searchPages(query)
		if err == nil {
			pages = searchResults
		}
	}

	searchContent := searchtemplates.SearchContent(query, pages)
	component := components.Page("Search", searchContent)
	component.Render(c.Request.Context(), c.Writer)
}

func searchPages(query string) ([]utils.PageInfoPrev, error) {
	searchResp, err := http.Get(fmt.Sprintf("%s/search?q=%s", config.SearchURL, url.QueryEscape(query)))
	if err != nil {
		return nil, err
	}
	defer searchResp.Body.Close()

	searchBody, err := io.ReadAll(searchResp.Body)
	if err != nil {
		log.Printf("Error reading search response body: %v", err)
		return nil, err
	}

	var searchResponse SearchResponse
	err = json.Unmarshal(searchBody, &searchResponse)
	if err != nil {
		log.Printf("Error unmarshaling search response: %v, body: %s", err, string(searchBody))
		return nil, err
	}

	if len(searchResponse.Results) == 0 {
		return []utils.PageInfoPrev{}, nil
	}

	slugsParam := strings.Join(searchResponse.Results, ",")
	wikiResp, err := http.Get(fmt.Sprintf("%s/pages?slugs=%s", config.WikiURL, url.QueryEscape(slugsParam)))
	if err != nil {
		return nil, err
	}
	defer wikiResp.Body.Close()

	wikiBody, err := io.ReadAll(wikiResp.Body)
	if err != nil {
		return nil, err
	}

	var pages []utils.PageInfoPrev
	err = json.Unmarshal(wikiBody, &pages)
	if err != nil {
		return nil, err
	}

	return pages, nil
}
