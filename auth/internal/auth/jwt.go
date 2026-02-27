package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID `json:"sub"`
	Email  string    `json:"email"`
	Roles  []string  `json:"roles"`
	jwt.RegisteredClaims
}

// JWTService handles JWT operations
type JWTService struct {
	secret     []byte
	expiration time.Duration
	issuer     string
	audience   string
}

// NewJWTService creates a new JWT service
func NewJWTService(secret string, expHours int) *JWTService {
	if expHours <= 0 {
		expHours = 24
	}

	return &JWTService{
		secret:     []byte(secret),
		expiration: time.Duration(expHours) * time.Hour,
		issuer:     "trevecca-pedia-auth",
		audience:   "trevecca-pedia",
	}
}

// GenerateToken generates a JWT token for a user
func (j *JWTService) GenerateToken(userID uuid.UUID, email string, roles []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Audience:  jwt.ClaimStrings{j.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(j.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	}, jwt.WithExpirationRequired()) // reject tokens that have no exp claim

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Verify issuer and audience
	if claims.Issuer != j.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	if len(claims.Audience) == 0 || claims.Audience[0] != j.audience {
		return nil, fmt.Errorf("invalid audience")
	}

	// Expiration is validated by ParseWithClaims + WithExpirationRequired above.
	// Do NOT add a manual "if claims.ExpiresAt != nil" check — the != nil guard
	// silently accepts tokens with no exp claim.

	return claims, nil
}
