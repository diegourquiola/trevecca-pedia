package auth

import (
	"api-layer/config"
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// proxyClient is used for all calls to the auth service.
// A 10-second timeout prevents goroutine leaks if the auth service hangs.
var proxyClient = &http.Client{Timeout: 10 * time.Second}

// PostLogin proxies POST /v1/auth/login → auth service POST /auth/login
func PostLogin(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read request"})
		return
	}

	res, err := proxyClient.Post(config.AuthServiceURL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
		return
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), respBody)
}

// PostRegister proxies POST /v1/auth/register → auth service POST /auth/register
func PostRegister(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read request"})
		return
	}

	res, err := proxyClient.Post(config.AuthServiceURL+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
		return
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), respBody)
}

// GetMe proxies GET /v1/auth/me → auth service GET /auth/me
// The Authorization header is forwarded so the auth service can validate the JWT.
func GetMe(c *gin.Context) {
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthServiceURL+"/auth/me", nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	res, err := proxyClient.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth service unavailable"})
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(res.StatusCode, res.Header.Get("Content-Type"), body)
}
