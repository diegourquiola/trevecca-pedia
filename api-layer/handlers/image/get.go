package image

import (
	"api-layer/config"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func GetImage(c *gin.Context) {
	id := strings.TrimPrefix(c.Param("id"), "/")

	res, err := httpClient.Get(fmt.Sprintf("%s/%s", config.ImageServiceURL, id))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch image."})
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		c.AbortWithStatusJSON(res.StatusCode, gin.H{"error": "Image not found."})
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response."})
		return
	}

	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Data(http.StatusOK, contentType, body)
}
