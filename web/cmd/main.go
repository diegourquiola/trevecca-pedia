package main

import (
	"web/auth"
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
	r.GET("/login", auth.GetLoginPage)
	r.GET("/profile", auth.GetProfilePage)

	port := config.GetEnv("WEB_SERVICE_PORT", "8080")
	r.Run(":" + port)
}
