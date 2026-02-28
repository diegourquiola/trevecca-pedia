package config

import (
	"os"

	"github.com/joho/godotenv"
)

var WikiServiceURL string
var SearchServiceURL string
var AuthServiceURL string

func init() {
	godotenv.Load()
	WikiServiceURL = GetEnv("WIKI_SERVICE_URL", "http://127.0.0.1:9454")
	SearchServiceURL = GetEnv("SEARCH_SERVICE_URL", "http://127.0.0.1:7724")
	AuthServiceURL = GetEnv("AUTH_SERVICE_URL", "http://127.0.0.1:8083")
}

// Note: To use external URLs for auto-start functionality, set these env vars:
// WIKI_SERVICE_URL=https://trevecca-pedia-wiki.fly.dev
// SEARCH_SERVICE_URL=https://trevecca-pedia-search.fly.dev

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetJWTSecret() string {
	return GetEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production")
}
