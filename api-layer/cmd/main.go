package main

import (
	"api-layer/config"
	"api-layer/handlers/search"
	"api-layer/handlers/wiki"
	"fmt"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.SetTrustedProxies(nil)
	gin.SetMode(gin.DebugMode)

	r.GET("/v1/wiki/pages", wiki.GetPages)
	r.GET("/v1/wiki/pages/:id", wiki.GetPage)
	r.GET("/v1/wiki/pages/:id/revisions", wiki.GetPageRevisions)
	r.GET("/v1/wiki/pages/:id/revisions/:rev", wiki.GetPageRevision)
	r.GET("/v1/wiki/indexable-pages", wiki.GetIndexablePages)

	r.POST("/v1/wiki/pages/new", wiki.PostNewPage)
	r.POST("/v1/wiki/pages/:id/delete", wiki.PostDeletePage)
	r.POST("/v1/wiki/pages/:id/revisions", wiki.PostPageRevision)

	r.GET("/v1/search/search", search.SearchRequest)

	port := config.GetEnv("API_LAYER_PORT", "2745")
	r.Run(fmt.Sprintf(":%s", port))
}
