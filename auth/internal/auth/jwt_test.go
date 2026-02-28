package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateToken(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, 24)

	userID := uuid.New()
	email := "test@example.com"
	roles := []string{"reader", "contributor"}

	token, err := jwtService.GenerateToken(userID, email, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Generated token is empty")
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, 24)

	userID := uuid.New()
	email := "test@example.com"
	roles := []string{"reader", "contributor"}

	token, err := jwtService.GenerateToken(userID, email, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %v, got %v", userID, claims.UserID)
	}

	if claims.Email != email {
		t.Errorf("Expected email %v, got %v", email, claims.Email)
	}

	if len(claims.Roles) != len(roles) {
		t.Errorf("Expected %d roles, got %d", len(roles), len(claims.Roles))
	}

	if claims.Issuer != "trevecca-pedia-auth" {
		t.Errorf("Expected issuer 'trevecca-pedia-auth', got %v", claims.Issuer)
	}

	if len(claims.Audience) == 0 || claims.Audience[0] != "trevecca-pedia" {
		t.Errorf("Expected audience 'trevecca-pedia', got %v", claims.Audience)
	}
}

func TestValidateTokenInvalidSecret(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, 24)

	userID := uuid.New()
	email := "test@example.com"
	roles := []string{"reader"}

	token, err := jwtService.GenerateToken(userID, email, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with different secret
	wrongService := NewJWTService("wrong-secret", 24)
	_, err = wrongService.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating with wrong secret")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	secret := "test-secret-key"
	// Create service with very short expiration
	jwtService := &JWTService{
		secret:     []byte(secret),
		expiration: 1 * time.Millisecond, // 1ms expiration
		issuer:     "trevecca-pedia-auth",
		audience:   "trevecca-pedia",
	}

	userID := uuid.New()
	email := "test@example.com"
	roles := []string{"reader"}

	token, err := jwtService.GenerateToken(userID, email, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Create new service with normal expiration for validation
	validationService := NewJWTService(secret, 24)
	_, err = validationService.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating expired token")
	}
}

func TestValidateTokenInvalidFormat(t *testing.T) {
	secret := "test-secret-key"
	jwtService := NewJWTService(secret, 24)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "invalid format",
			token: "not.a.valid.jwt",
		},
		{
			name:  "random string",
			token: "randomstring",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := jwtService.ValidateToken(tt.token)
			if err == nil {
				t.Errorf("Expected error for invalid token")
			}
		})
	}
}

func TestNewJWTService(t *testing.T) {
	tests := []struct {
		name     string
		secret   string
		expHours int
		wantExp  time.Duration
	}{
		{
			name:     "valid config",
			secret:   "test-secret",
			expHours: 24,
			wantExp:  24 * time.Hour,
		},
		{
			name:     "zero expiration defaults to 24h",
			secret:   "test-secret",
			expHours: 0,
			wantExp:  24 * time.Hour,
		},
		{
			name:     "negative expiration defaults to 24h",
			secret:   "test-secret",
			expHours: -1,
			wantExp:  24 * time.Hour,
		},
		{
			name:     "custom expiration",
			secret:   "test-secret",
			expHours: 48,
			wantExp:  48 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewJWTService(tt.secret, tt.expHours)
			if service.expiration != tt.wantExp {
				t.Errorf("Expected expiration %v, got %v", tt.wantExp, service.expiration)
			}
		})
	}
}
