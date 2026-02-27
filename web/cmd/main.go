package main

import (
	"web/config"
	"web/handlers/image"
	"web/handlers/search"
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

	r.GET("/image/*id", image.GetImage)

	port := config.GetEnv("WEB_SERVICE_PORT", "8080")
	r.Run(":" + port)
}
