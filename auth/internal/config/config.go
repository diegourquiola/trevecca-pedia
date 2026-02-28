package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Port        string
	DBHost      string
	DBPort      string
	DBName      string
	DBUser      string
	DBPassword  string
	DBURL       string
	JWTSecret   string
	JWTExpHours int
	CORSOrigins []string
	DevSeed     bool
}

// DatabaseURL builds a PostgreSQL keyword/value connection string from the
// individual DB fields, matching the pattern used by the wiki service.
func (c Config) DatabaseURL() string {
	if c.DBURL != "" {
		return c.DBURL
	}
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		c.DBHost, c.DBPort, c.DBName, c.DBUser, c.DBPassword)
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	config := &Config{
		Port:        getEnv("PORT", "8083"),
		DBURL:       getEnv("DATABASE_URL", ""),
		DBHost:      getEnv("AUTH_DB_HOST", "localhost"),
		DBPort:      getEnv("AUTH_DB_PORT", "5433"),
		DBName:      getEnv("AUTH_DB_NAME", "auth"),
		DBUser:      getEnv("AUTH_DB_USER", "auth_user"),
		DBPassword:  getEnv("AUTH_DB_PASSWORD", "change_me_in_production"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
		JWTExpHours: getEnvInt("JWT_EXP_HOURS", 24),
		CORSOrigins: getEnvSlice("CORS_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
		DevSeed:     getEnvBool("DEV_SEED", false),
	}

	return config, nil
}

// String redacts sensitive fields so Config can be safely logged.
func (c Config) String() string {
	return fmt.Sprintf(
		"{Port:%s DBHost:%s DBPort:%s DBName:%s DBUser:%s DBPassword:[REDACTED] JWTSecret:[REDACTED] JWTExpHours:%d CORSOrigins:%v DevSeed:%t}",
		c.Port, c.DBHost, c.DBPort, c.DBName, c.DBUser, c.JWTExpHours, c.CORSOrigins, c.DevSeed,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}
