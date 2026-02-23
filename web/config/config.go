package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var WikiURL string
var SearchURL string
var AuthURL string

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	apiURL := GetEnv("API_LAYER_URL", "http://127.0.0.1:2745/v1")
	WikiURL = fmt.Sprintf("%s/wiki", apiURL)
	SearchURL = fmt.Sprintf("%s/search", apiURL)
	AuthURL = GetEnv("AUTH_SERVICE_URL", "http://127.0.0.1:8083")
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
