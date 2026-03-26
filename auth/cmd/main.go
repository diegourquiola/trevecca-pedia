package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"auth/internal/auth"
	"auth/internal/config"
	httphandler "auth/internal/http"
	"auth/internal/store"

	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connection established")

	// Initialize store
	dataStore := store.NewStore(db)

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpHours)

	// Seed dev user if configured
	if cfg.DevSeed {
		log.Println("⚠️  DEV_SEED is enabled - creating development users")
		if err := seedDevUser(context.Background(), dataStore); err != nil {
			log.Printf("Warning: Failed to seed dev user: %v", err)
		} else {
			log.Println("✓ Development user created/verified: dev@trevecca.edu / devpass")
		}
		if err := seedModUser(context.Background(), dataStore); err != nil {
			log.Printf("Warning: Failed to seed mod user: %v", err)
		} else {
			log.Println("✓ Development user created/verified: mod@trevecca.edu / modpass")
		}
	}

	// Setup router
	router := httphandler.SetupRouter(dataStore, jwtService, cfg.CORSOrigins)

	// Start server
	log.Printf("Starting auth service on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func seedDevUser(ctx context.Context, dataStore *store.Store) error {
	// Check if dev user already exists
	_, err := dataStore.GetUserByEmail(ctx, "dev@trevecca.edu")
	if err == nil {
		// User already exists
		return nil
	}

	// Hash password
	hashedPassword, err := auth.HashPassword("devpass")
	if err != nil {
		return err
	}

	// Create user
	user, err := dataStore.CreateUser(ctx, "dev@trevecca.edu", hashedPassword)
	if err != nil {
		return err
	}

	// Get contributor role
	role, err := dataStore.GetRoleByName(ctx, "contributor")
	if err != nil {
		return err
	}

	// Add contributor role to user
	return dataStore.AddUserRole(ctx, user.ID, role.ID)
}

func seedModUser(ctx context.Context, dataStore *store.Store) error {
	_, err := dataStore.GetUserByEmail(ctx, "mod@trevecca.edu")
	if err == nil {
		return nil
	}
	hashedPassword, err := auth.HashPassword("modpass")
	if err != nil {
		return err
	}
	user, err := dataStore.CreateUser(ctx, "mod@trevecca.edu", hashedPassword)
	if err != nil {
		return err
	}
	contributorRole, err := dataStore.GetRoleByName(ctx, "contributor")
	if err != nil {
		return err
	}
	moderatorRole, err := dataStore.GetRoleByName(ctx, "moderator")
	if err != nil {
		return err
	}
	err = dataStore.AddUserRole(ctx, user.ID, contributorRole.ID)
	if err != nil {
		return err
	}
	return dataStore.AddUserRole(ctx, user.ID, moderatorRole.ID)
}
