package config

import (
	"os"

	"github.com/joho/godotenv"
)

var WikiServiceURL string
var SearchServiceURL string
var ImageServiceURL string

func init() {
	godotenv.Load()
	WikiServiceURL = GetEnv("WIKI_SERVICE_URL", "http://127.0.0.1:9454")
	SearchServiceURL = GetEnv("SEARCH_SERVICE_URL", "http://127.0.0.1:7724")
	ImageServiceURL = GetEnv("IMAGE_SERVICE_URL", "https://treveccabuddy.tp-images.workers.dev")
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
