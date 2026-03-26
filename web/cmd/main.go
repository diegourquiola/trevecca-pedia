package main

import (
	"web/auth"
	"web/config"
	"web/handlers/image"
	"web/handlers/search"
	"web/users"
	"web/wiki"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.SetTrustedProxies(nil)
	gin.SetMode(gin.DebugMode)

	r.Static("./static", "static")
	r.GET("/", wiki.GetHome)
	r.GET("/pages", wiki.GetCategoryPages)
	r.GET("/pages/:id", wiki.GetPage)
	r.GET("/pages/:id/history", wiki.GetPageHistory)
	r.GET("/pages/:id/history/:revId", wiki.GetPageHistory)
	r.GET("/pages/:id/history/timeline", wiki.GetTimelinePartial)
	r.GET("/search", search.GetSearchPage)
	r.GET("/login", auth.GetLoginPage)
	r.GET("/users/:username", users.GetUserProfilePage)
	r.GET("/users/:username/revisions", users.GetUserRevisionsPartial)

	// Auth API proxy routes - browser calls these, web service forwards to API layer
	r.POST("/auth/login", auth.PostLogin)
	r.POST("/auth/register", auth.PostRegister)
	r.GET("/auth/me", auth.GetMe)
	r.POST("/auth/logout", auth.PostLogout)

	// Protected editing routes - require authentication
	protected := r.Group("/")
	protected.Use(auth.RequireAuth())
	{
		protected.GET("/pages/new", wiki.GetCreatePage)
		protected.POST("/pages/new", wiki.PostCreatePage)
		protected.GET("/pages/:id/edit", wiki.GetEditPage)
		protected.POST("/pages/:id/edit", wiki.PostEditPage)
		protected.POST("/update-preview", wiki.PostPreview)
	}

	// Moderator-only routes - require moderator role
	moderator := r.Group("/")
	moderator.Use(auth.RequireRole("moderator"))
	{
		moderator.POST("/pages/:id/delete", wiki.PostDeletePage)
	}

	r.GET("/image/*id", image.GetImage)

	port := config.GetEnv("WEB_SERVICE_PORT", "8080")
	r.Run(":" + port)
}
