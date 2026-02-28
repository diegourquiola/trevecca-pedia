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

	// Auth API proxy routes - browser calls these, web service forwards to API layer
	r.POST("/auth/login", auth.PostLogin)
	r.POST("/auth/register", auth.PostRegister)
	r.GET("/auth/me", auth.GetMe)
	r.POST("/auth/logout", auth.PostLogout)

	port := config.GetEnv("WEB_SERVICE_PORT", "8080")
	r.Run(":" + port)
}
