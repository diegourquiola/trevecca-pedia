package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"api-layer/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims - matches auth service structure
type Claims struct {
	UserID uuid.UUID `json:"sub"`
	Email  string    `json:"email"`
	Roles  []string  `json:"roles"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Store claims in context for handlers to use
		c.Set("claims", claims)
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// RequireRole middleware checks if user has a specific role
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesValue, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "no roles found"})
			c.Abort()
			return
		}

		roles, ok := rolesValue.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid roles format"})
			c.Abort()
			return
		}

		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		c.Abort()
	}
}

func validateToken(tokenString string) (*Claims, error) {
	secret := config.GetJWTSecret()

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithExpirationRequired()) // reject tokens that have no exp claim

	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Verify issuer and audience
	if claims.Issuer != "trevecca-pedia-auth" {
		return nil, fmt.Errorf("invalid issuer")
	}

	if len(claims.Audience) == 0 || claims.Audience[0] != "trevecca-pedia" {
		return nil, fmt.Errorf("invalid audience")
	}

	// Expiration validated by ParseWithClaims + WithExpirationRequired above.
	return claims, nil
}
