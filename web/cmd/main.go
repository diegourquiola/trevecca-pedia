package main

import (
	"web/auth"
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
	r.GET("/pages/new", wiki.GetCreatePage)
	r.POST("/pages/new", wiki.PostCreatePage)
	r.GET("/pages/:id", wiki.GetPage)
	r.GET("/pages/:id/edit", wiki.GetEditPage)
	r.POST("/pages/:id/edit", wiki.PostEditPage)
	r.GET("/search", search.GetSearchPage)
	r.GET("/login", auth.GetLoginPage)
	r.GET("/profile", auth.GetProfilePage)

	// Auth API proxy routes - browser calls these, web service forwards to API layer
	r.POST("/auth/login", auth.PostLogin)
	r.POST("/auth/register", auth.PostRegister)
	r.GET("/auth/me", auth.GetMe)
	r.POST("/auth/logout", auth.PostLogout)

	r.GET("/image/*id", image.GetImage)

	port := config.GetEnv("WEB_SERVICE_PORT", "8080")
	r.Run(":" + port)
}
