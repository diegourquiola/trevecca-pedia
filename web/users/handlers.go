package users

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"web/config"
	"web/templates/components"
	"web/templates/users"

	"github.com/gin-gonic/gin"
)

// UserResponse represents the user data from auth service
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

// RevisionResponse represents a revision from wiki service
type RevisionResponse struct {
	UUID        string     `json:"uuid"`
	PageId      *string    `json:"page_id"`
	DateTime    *time.Time `json:"date_time"`
	Author      *string    `json:"author"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	ArchiveDate *time.Time `json:"archive_date"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

// httpClient with timeout
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

// GetUserProfilePage handles GET /users/:username
func GetUserProfilePage(c *gin.Context) {
	username := c.Param("username")

	if username == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}

	// Fetch user data from auth service
	user, err := fetchUserByUsername(username)
	if err != nil {
		// Check if it's a 404 (user not found)
		if err.Error() == "user not found" {
			// Render user not found page with HTTP 200
			c.Header("Content-Type", "text/html")
			notFoundContent := users.UserNotFoundContent(username)
			page := components.Page("User Not Found", notFoundContent)
			if err := page.Render(context.Background(), c.Writer); err != nil {
				log.Printf("error rendering user not found page: %v", err)
				c.String(http.StatusInternalServerError, "Internal Server Error")
			}
			return
		}
		log.Printf("error fetching user %s: %v", username, err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}


	// Fetch initial revisions (first 20) - pass only the username part of email
	revisions, hasMore, err := fetchRevisionsByAuthor(username, 0, 20)
	if err != nil {
		log.Printf("error fetching revisions for user %s: %v", username, err)
		// Continue with empty revisions
		revisions = []users.Revision{}
		hasMore = false
	}


	// Build profile user data
	profileUser := users.ProfileUser{
		ID:        user.ID,
		Email:     user.Email,
		Username:  username,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt,
	}


	// Render profile page using the same pattern as wiki handlers
	c.Header("Content-Type", "text/html")
	profileContent := users.ProfileContent(profileUser, revisions, hasMore)
	page := components.Page(profileUser.Username+"'s Profile", profileContent)
	if err := page.Render(context.Background(), c.Writer); err != nil {
		log.Printf("error rendering profile page: %v", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

}

// GetUserRevisionsPartial handles GET /users/:username/revisions for HTMX infinite scroll
func GetUserRevisionsPartial(c *gin.Context) {
	username := c.Param("username")
	offsetStr := c.DefaultQuery("offset", "20")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 20
	}

	// Fetch revisions with offset - use username directly (API layer appends @trevecca.edu)
	revisions, hasMore, err := fetchRevisionsByAuthor(username, offset, 20)
	if err != nil {
		log.Printf("error fetching revisions for user %s: %v", username, err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Render revisions partial
	content := users.RevisionsPartial(revisions, username, offset+len(revisions), hasMore)
	if err := content.Render(c.Request.Context(), c.Writer); err != nil {
		log.Printf("error rendering revisions partial: %v", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}

// fetchUserByUsername fetches user data from auth service
func fetchUserByUsername(username string) (*UserResponse, error) {
	url := fmt.Sprintf("%s/users/%s", config.AuthURL, username)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth service returned %d: %s", resp.StatusCode, string(body))
	}

	var user UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}

	return &user, nil
}

// fetchRevisionsByAuthor fetches revisions from wiki service
func fetchRevisionsByAuthor(author string, offset, limit int) ([]users.Revision, bool, error) {
	url := fmt.Sprintf("%s/revisions?author=%s&index=%d&count=%d", config.WikiURL, url.QueryEscape(author), offset, limit)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch revisions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("wiki service returned %d: %s", resp.StatusCode, string(body))
	}

	var revs []RevisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&revs); err != nil {
		return nil, false, fmt.Errorf("failed to decode revisions response: %w", err)
	}

	// Convert to template format
	var result []users.Revision
	for _, rev := range revs {
		revision := users.Revision{
			UUID:     rev.UUID,
			PageSlug: rev.Slug,
			PageName: rev.Name,
		}

		if rev.PageId != nil {
			revision.PageId = *rev.PageId
		}
		if rev.DateTime != nil {
			revision.DateTime = *rev.DateTime
		}
		if rev.Author != nil {
			revision.Author = *rev.Author
		}

		result = append(result, revision)
	}

	// Check if there are more revisions
	hasMore := len(result) == limit

	return result, hasMore, nil
}

// GetCurrentUserProfile redirects /profile to the current user's profile page
func GetCurrentUserProfile(c *gin.Context) {
	// Get current user from auth cookie
	// This is a placeholder - in reality, you'd extract the user from the cookie/session
	// For now, redirect to home
	c.Redirect(http.StatusFound, "/")
}

// ExtractUsername extracts username from email (part before @)
func ExtractUsername(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}
