package utils

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	wikierrors "wiki/errors"

	"github.com/joho/godotenv"
)

// init loads the .env file when the package is imported
func init() {
	// Only load .env file in local development (not on fly.io)
	// Fly.io sets FLY_APP_NAME environment variable
	if os.Getenv("FLY_APP_NAME") == "" {
		// Try to load .env file, but don't fail if it doesn't exist
		// This allows environment variables to be set via other means (Docker, shell, etc.)
		_ = godotenv.Load(".env")
	}
}

func GetDatabase() (*sql.DB, error) {
	// Get connection parameters from environment variables with defaults

	var host, port, dbname, user, password string
	databaseUrl := getEnv("DATABASE_URL", "")
	if databaseUrl == "" {
		host = getEnv("WIKI_DB_HOST", "localhost")
		port = getEnv("WIKI_DB_PORT", "5432")
		dbname = getEnv("WIKI_DB_NAME", "wiki")
		user = getEnv("WIKI_DB_USER", "wiki_user")
		password = getEnv("WIKI_DB_PASSWORD", "myatt")
	} else {
		host, port, dbname, user, password = parseDatabaseURL(databaseUrl)
	}

	connStr := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		host, port, dbname, user, password)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to connect to database: %s\n", err)
		return nil, wikierrors.DatabaseError(err)
	}
	//log.Printf("Connected to database: %s\n", connStr)
	return db, nil
}

// getEnv retrieves an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseDatabaseURL parses a PostgreSQL connection URL and returns connection parameters
// Expected format: postgres://user:password@host:port/dbname or postgresql://user:password@host:port/dbname
func parseDatabaseURL(databaseURL string) (host, port, dbname, user, password string) {
	// Set defaults
	host = "localhost"
	port = "5432"

	u, err := url.Parse(databaseURL)
	if err != nil {
		log.Printf("Failed to parse DATABASE_URL: %s\n", err)
		return
	}

	if u.Hostname() != "" {
		host = u.Hostname()
	}

	if u.Port() != "" {
		port = u.Port()
	}

	if u.Path != "" {
		dbname = strings.TrimPrefix(u.Path, "/")
	}

	if u.User != nil {
		user = u.User.Username()
		if pass, ok := u.User.Password(); ok {
			password = pass
		}
	}

	return
}

func GetDataDir() string {
	if dataDir := os.Getenv("WIKI_DATA_DIR"); dataDir != "" {
		//log.Printf("dataDir: %s\n", dataDir)
		return dataDir
	}
	return filepath.Join("..", "wiki-fs")
}

