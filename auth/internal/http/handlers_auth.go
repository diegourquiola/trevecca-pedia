package http

import (
	"log"
	"net/http"
	"strings"

	"auth/internal/auth"
	"auth/internal/store"

	"github.com/gin-gonic/gin"
)

const allowedEmailDomain = "@trevecca.edu"

// AuthHandlers handles authentication endpoints
type AuthHandlers struct {
	store      *store.Store
	jwtService *auth.JWTService
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(store *store.Store, jwtService *auth.JWTService) *AuthHandlers {
	return &AuthHandlers{
		store:      store,
		jwtService: jwtService,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken string               `json:"accessToken"`
	User        *store.UserWithRoles `json:"user"`
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// Login handles user login
func (h *AuthHandlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	// Get user by email
	user, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("Login failed for %s: %v", req.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		log.Printf("Login failed for %s: invalid password", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Get user roles
	roles, err := h.store.GetUserRoles(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting roles for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user.ID, user.Email, roles)
	if err != nil {
		log.Printf("Error generating token for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("User %s logged in successfully", user.Email)

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken: token,
		User: &store.UserWithRoles{
			ID:    user.ID,
			Email: user.Email,
			Roles: roles,
		},
	})
}

// Register handles user registration
func (h *AuthHandlers) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format - email required and password must be at least 8 characters"})
		return
	}

	// Normalize email to lowercase for consistent DB lookups
	req.Email = strings.ToLower(req.Email)

	// Validate email domain
	if !strings.HasSuffix(req.Email, allowedEmailDomain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only @trevecca.edu email addresses are allowed"})
		return
	}

	// Check email whitelist
	whitelisted, err := h.store.IsEmailWhitelisted(c.Request.Context(), req.Email)
	if err != nil {
		log.Printf("Error checking whitelist for %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if !whitelisted {
		log.Printf("Registration blocked for non-whitelisted email: %s", req.Email)
		c.JSON(http.StatusForbidden, gin.H{"error": "this email is not authorized to register"})
		return
	}

	// Check if user already exists
	existingUser, err := h.store.GetUserByEmail(c.Request.Context(), req.Email)
	if err == nil && existingUser != nil {
		// err == nil means the user was found (no error) — it's a duplicate
		c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
		return
	}
	// err != nil here means "user not found" (expected) or a transient DB error;
	// the subsequent RegisterUser call will surface any real DB failure.

	// Hash password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Create user and assign contributor role atomically in one transaction
	user, roles, err := h.store.RegisterUser(c.Request.Context(), req.Email, hashedPassword, "contributor")
	if err != nil {
		log.Printf("Error registering user %s: %v", req.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Generate JWT token so user is logged in after registration
	token, err := h.jwtService.GenerateToken(user.ID, user.Email, roles)
	if err != nil {
		log.Printf("Error generating token for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Printf("User %s registered successfully", user.Email)

	c.JSON(http.StatusCreated, LoginResponse{
		AccessToken: token,
		User: &store.UserWithRoles{
			ID:    user.ID,
			Email: user.Email,
			Roles: roles,
		},
	})
}

// Me handles getting current user info
func (h *AuthHandlers) Me(c *gin.Context) {
	// Get claims from context (set by AuthMiddleware)
	claimsValue, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	claims, ok := claimsValue.(*auth.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Return user info from claims
	c.JSON(http.StatusOK, store.UserWithRoles{
		ID:    claims.UserID,
		Email: claims.Email,
		Roles: claims.Roles,
	})
}

// User handles getting user info by username
func (h *AuthHandlers) User(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username required"})
		return
	}

	// Get user by username (email prefix)
	user, err := h.store.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		log.Printf("Error getting user %s: %v", username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	// Get user roles
	roles, err := h.store.GetUserRoles(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Error getting roles for user %s: %v", username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, store.UserWithRoles{
		ID:        user.ID,
		Email:     user.Email,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
	})
}

// HealthCheck handles health check endpoint
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
