package wiki

import (
	"api-layer/config"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetPages(c *gin.Context) {
	catQuery := c.DefaultQuery("category", "")
	slugsQuery := c.DefaultQuery("slugs", "")
	ind, err := strconv.Atoi(c.DefaultQuery("index", "0"))
	if err != nil {
		ind = 0
	}
	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil {
		count = 10
	}
	exact := c.DefaultQuery("exact", "false")

	res, err := http.Get(fmt.Sprintf("%s/pages?category=%s&slugs=%s&index=%d&count=%d&exact=%s",
		config.WikiServiceURL, catQuery, slugsQuery, ind, count, exact))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pages."})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}

func GetPage(c *gin.Context) {
	id := c.Param("id")
	res, err := http.Get(fmt.Sprintf("%s/pages/%s", config.WikiServiceURL, id))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pages."})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}

func GetPageRevisions(c *gin.Context) {
	id := c.Param("id")
	ind, err := strconv.Atoi(c.DefaultQuery("index", "0"))
	if err != nil {
		ind = 0
	}
	count, err := strconv.Atoi(c.DefaultQuery("count", "10"))
	if err != nil {
		count = 10
	}

	res, err := http.Get(fmt.Sprintf("%s/pages/%s/revisions?index=%d&count=%d",
		config.WikiServiceURL, id, ind, count))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pages."})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}

func GetPageRevision(c *gin.Context) {
	id := c.Param("id")
	revId := c.Param("rev")
	res, err := http.Get(fmt.Sprintf("%s/pages/%s/revisions/%s", config.WikiServiceURL, id, revId))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pages."})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}

func GetIndexablePages(c *gin.Context) {
	ind := c.DefaultQuery("index", "0")
	count := c.DefaultQuery("count", "10")
	resp, err := http.Get(fmt.Sprintf("%s/indexable-pages?index=%s&count=%s", config.WikiServiceURL, ind, count))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pages."})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

func GetCategories(c *gin.Context) {
	tree := c.DefaultQuery("tree", "false")
	root := c.DefaultQuery("root", "false")

	res, err := http.Get(fmt.Sprintf("%s/categories?tree=%s&root=%s",
		config.WikiServiceURL, tree, root))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories."})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}

func GetPageCategories(c *gin.Context) {
	id := c.Param("id")
	res, err := http.Get(fmt.Sprintf("%s/pages/%s/categories", config.WikiServiceURL, id))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch page categories."})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}
