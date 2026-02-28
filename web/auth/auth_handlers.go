package auth

import (
	"log"
	"web/templates/auth"
	"web/templates/components"

	"github.com/gin-gonic/gin"
)

func GetLoginPage(c *gin.Context) {
	content := auth.AuthPage()
	page := components.Page("Log In", content)
	if err := page.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("error rendering login page: %v", err)
	}
}

func GetProfilePage(c *gin.Context) {
	content := auth.ProfilePage()
	page := components.Page("Profile", content)
	if err := page.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("error rendering profile page: %v", err)
	}
}
