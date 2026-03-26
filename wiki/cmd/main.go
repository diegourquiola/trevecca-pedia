package main

import (
	"os"
	"wiki/handlers"
	"wiki/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// utils import triggers .env loading via init()
	// Ensure database is accessible
	_, err := utils.GetDatabase()
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)
	gin.SetMode(gin.DebugMode)

	// GET

	// /pages?category={cat}&index={ind}&count={count}
	r.GET("/pages", handlers.PagesHandler)

	r.GET("/pages/:id", handlers.PageHandler)

	r.GET("/revisions", handlers.PageRevisionsHandler)

	r.GET("/pages/:id/revisions", handlers.PageRevisionsHandler)

	r.GET("/pages/:id/revisions/:rev", handlers.PageRevisionHandler)

	r.GET("/indexable-pages", handlers.IndexablePagesHandler)

	r.GET("/categories", handlers.CategoriesHandler)

	r.GET("/pages/:id/categories", handlers.GetPageCategoriesHandler)

	// POST

	r.POST("/pages/new", handlers.NewPageHandler)

	r.POST("/pages/:id/delete", handlers.DeletePageHandler)

	r.POST("/pages/:id/revisions", handlers.NewRevisionHandler)

	r.POST("/pages/:id/categories", handlers.SetPageCategoriesHandler) // Requires auth

	// Use port from environment variable, default to 9454
	port := os.Getenv("WIKI_SERVICE_PORT")
	if port == "" {
		port = "9454"
	}
	r.Run(":" + port)
}
