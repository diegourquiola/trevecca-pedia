package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	"web/config"

	"github.com/gin-gonic/gin"
)

const authCookieName = "auth_token"

// proxyClient is used for all calls to the upstream API layer.
// A 10-second timeout prevents goroutine leaks if the upstream service hangs.
var proxyClient = &http.Client{Timeout: 10 * time.Second}

// authServiceResponse is the shape of a successful login/register response from the API layer.
type authServiceResponse struct {
	AccessToken string          `json:"accessToken"`
	User        json.RawMessage `json:"user"`
}

// setAuthCookie writes the JWT as an HttpOnly cookie on the response.
// HttpOnly prevents JavaScript from reading the token, blocking XSS-based token theft.
// Set COOKIE_SECURE=true in production (HTTPS) environments.
func setAuthCookie(c *gin.Context, token string) {
	secure := config.GetEnv("COOKIE_SECURE", "false") == "true"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}

// clearAuthCookie expires the auth cookie, effectively logging the user out server-side.
func clearAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

// handleAuthPost proxies a POST to the API layer, intercepts the token from the
// success response, sets it as an HttpOnly cookie, and returns only {user: ...} to the
// browser. Error responses are passed through unchanged.
func handleAuthPost(c *gin.Context, upstreamPath string) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read request"})
		return
	}

	res, err := proxyClient.Post(config.AuthURL+upstreamPath, "application/json", bytes.NewReader(body))
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

	// On success, extract the token, set it as HttpOnly cookie, return only user data.
	if res.StatusCode == http.StatusOK || res.StatusCode == http.StatusCreated {
		var parsed authServiceResponse
		if err := json.Unmarshal(respBody, &parsed); err == nil && parsed.AccessToken != "" {
			setAuthCookie(c, parsed.AccessToken)
			c.Data(res.StatusCode, "application/json", mustMarshal(map[string]json.RawMessage{
				"user": parsed.User,
			}))
			return
		}
		// Couldn't extract token — log and fall through to pass the raw response
		log.Printf("warn: could not extract token from auth response at %s", upstreamPath)
	}

	// Error response — pass through as-is (400, 401, 403, 409, 500, etc.)
	c.Data(res.StatusCode, res.Header.Get("Content-Type"), respBody)
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// PostLogin proxies POST /auth/login → API layer POST /v1/auth/login
func PostLogin(c *gin.Context) {
	handleAuthPost(c, "/login")
}

// PostRegister proxies POST /auth/register → API layer POST /v1/auth/register
func PostRegister(c *gin.Context) {
	handleAuthPost(c, "/register")
}

// GetMe proxies GET /auth/me → API layer GET /v1/auth/me
// Reads the HttpOnly auth cookie and converts it to an Authorization header
// for the upstream API layer.
func GetMe(c *gin.Context) {
	tokenCookie, err := c.Cookie(authCookieName)
	if err != nil || tokenCookie == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, config.AuthURL+"/me", nil)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+tokenCookie)

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

// PostLogout clears the auth cookie, logging the user out server-side.
// Since the token is HttpOnly, only the server can remove it — JS logout()
// must call this endpoint before redirecting.
func PostLogout(c *gin.Context) {
	clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
