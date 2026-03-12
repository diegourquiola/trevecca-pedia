package auth

import (
	"net/http"
	"net/url"
	"web/config"

	"github.com/gin-gonic/gin"
)

// RequireAuth middleware checks if the user is authenticated.
// If not authenticated, it redirects to the login page with a redirect URL parameter.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the auth cookie exists
		tokenCookie, err := c.Cookie(authCookieName)
		if err != nil || tokenCookie == "" {
			// User is not authenticated - redirect to login with return URL
			redirectURL := url.QueryEscape(c.Request.URL.RequestURI())
			loginURL := "/login?redirect=" + redirectURL
			c.Redirect(http.StatusFound, loginURL)
			c.Abort()
			return
		}

		// Try to validate the token by calling the auth service
		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthURL+"/me", nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
			return
		}
		req.Header.Set("Authorization", "Bearer "+tokenCookie)

		res, err := proxyClient.Do(req)
		if err != nil {
			// Auth service unavailable - be conservative and redirect to login
			redirectURL := url.QueryEscape(c.Request.URL.RequestURI())
			loginURL := "/login?redirect=" + redirectURL
			c.Redirect(http.StatusFound, loginURL)
			c.Abort()
			return
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			// Token is invalid or expired - redirect to login
			redirectURL := url.QueryEscape(c.Request.URL.RequestURI())
			loginURL := "/login?redirect=" + redirectURL
			c.Redirect(http.StatusFound, loginURL)
			c.Abort()
			return
		}

		// User is authenticated, proceed to the next handler
		c.Next()
	}
}
