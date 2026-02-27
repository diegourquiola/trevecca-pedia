package main

import (
	"net/http/httputil"
	"net/url"
	"web/config"
	"web/search"
	"web/wiki"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.SetTrustedProxies(nil)
	gin.SetMode(gin.DebugMode)

	r.Static("./static", "static")
	r.GET("/", wiki.GetHome)
	r.GET("/pages/:id", wiki.GetPage)
	r.GET("/search", search.GetSearchPage)

	// Proxy /image/* in markdown to api-layer
	apiURL, _ := url.Parse("http://localhost:2745")
	proxy := httputil.NewSingleHostReverseProxy(apiURL)

	r.GET("/image/*id", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	port := config.GetEnv("WEB_SERVICE_PORT", "8080")
	r.Run(":" + port)
}
