package images

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetImage(c *gin.Context) {
	filename := c.Param("filename")
	// needs to not be hardcoded, and not static and not stored in api-layer directory
	res, err := http.Get(fmt.Sprintf("%s/static/images/%s", "http://127.0.0.1:2745", filename))
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}
