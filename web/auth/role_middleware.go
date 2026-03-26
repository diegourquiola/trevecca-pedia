package auth

import (
	"encoding/json"
	"net/http"
	"web/config"

	"github.com/gin-gonic/gin"
)

// User represents the authenticated user from the auth service
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

// GetUserFromContext extracts user info from the auth cookie by calling /auth/me
func GetUserFromContext(c *gin.Context) (*User, error) {
	tokenCookie, err := c.Cookie(authCookieName)
	if err != nil || tokenCookie == "" {
		return nil, nil // Not authenticated
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthURL+"/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tokenCookie)

	res, err := proxyClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, nil // Invalid or expired token
	}

	var user User
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// HasRole checks if a user has a specific role
func HasRole(user *User, role string) bool {
	if user == nil {
		return false
	}
	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// RequireRole middleware ensures the user has the specified role
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := GetUserFromContext(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check user roles"})
			return
		}

		if user == nil {
			// Not authenticated - redirect to login
			redirectURL := c.Request.URL.RequestURI()
			c.Redirect(http.StatusFound, "/login?redirect="+redirectURL)
			c.Abort()
			return
		}

		if !HasRole(user, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		// Store user in context for handlers to use
		c.Set("user", user)
		c.Next()
	}
}
