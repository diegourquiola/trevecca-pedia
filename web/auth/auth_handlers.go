package auth

import (
	"log"
	"strings"
	"web/templates/auth"
	"web/templates/components"

	"github.com/gin-gonic/gin"
)

func GetLoginPage(c *gin.Context) {
	// Get the redirect URL from the query parameter, default to home page
	redirectURL := c.Query("redirect")
	if redirectURL == "" || !strings.HasPrefix(redirectURL, "/") || strings.HasPrefix(redirectURL, "//") {
		redirectURL = "/"
	}

	content := auth.AuthPage(redirectURL)
	page := components.Page("Log In", content)
	if err := page.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("error rendering login page: %v", err)
	}
}
