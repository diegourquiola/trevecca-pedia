package http

import (
	"auth/internal/auth"
	"auth/internal/store"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures all routes
func SetupRouter(store *store.Store, jwtService *auth.JWTService, corsOrigins []string) *gin.Engine {
	r := gin.Default()
	r.SetTrustedProxies(nil)

	// CORS middleware
	r.Use(CORSMiddleware(corsOrigins))

	// Health check
	r.GET("/healthz", HealthCheck)

	// Auth handlers
	authHandlers := NewAuthHandlers(store, jwtService)

	// Auth routes
	r.POST("/auth/register", authHandlers.Register)
	r.POST("/auth/login", authHandlers.Login)
	r.GET("/auth/me", AuthMiddleware(jwtService), authHandlers.Me)

	return r
}
