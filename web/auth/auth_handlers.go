package auth

import (
	"web/templates/auth"
	"web/templates/components"

	"github.com/gin-gonic/gin"
)

func GetLoginPage(c *gin.Context) {
	content := auth.AuthPage()
	page := components.Page("Log In", content)
	page.Render(c.Request.Context(), c.Writer)
}

func GetProfilePage(c *gin.Context) {
	content := auth.ProfilePage()
	page := components.Page("Profile", content)
	page.Render(c.Request.Context(), c.Writer)
}
