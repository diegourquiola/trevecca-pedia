package wiki

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"web/auth"
	"web/config"

	"github.com/gin-gonic/gin"
)

// PostDeletePage handles the delete form submission
func PostDeletePage(c *gin.Context) {
	id := c.Param("id")

	// Get user email from context (set by RequireRole middleware)
	userValue, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/")
		return
	}

	user, ok := userValue.(*auth.User)
	if !ok {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Build form data
	formData := url.Values{}
	formData.Set("slug", id)
	formData.Set("user", user.Email)

	// Forward to API layer
	req, err := http.NewRequestWithContext(
		c.Request.Context(),
		http.MethodPost,
		fmt.Sprintf("%s/pages/%s/delete", config.WikiURL, id),
		bytes.NewBufferString(formData.Encode()),
	)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Get auth token from cookie
	tokenCookie, err := c.Cookie("auth_token")
	if err == nil && tokenCookie != "" {
		req.Header.Set("Authorization", "Bearer "+tokenCookie)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Success - redirect to home page
		c.Redirect(http.StatusFound, "/?deleted=true")
		return
	}

	// Error - redirect back to page with error
	c.Redirect(http.StatusFound, fmt.Sprintf("/pages/%s?error=delete_failed", id))
}
