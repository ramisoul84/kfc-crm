package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
)

// TokenType distinguishes access tokens from refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims is the JWT payload.
type Claims struct {
	UserID    uuid.UUID   `json:"user_id"`
	Email     string      `json:"email,omitempty"`
	Role      domain.Role `json:"role,omitempty"`
	TokenType TokenType   `json:"token_type"`

	jwt.RegisteredClaims
}

// TokenManager signs and validates JWTs.
type TokenManager struct {
	secretKey       []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
	issuer          string
}

// NewTokenManager creates a TokenManager.
func NewTokenManager(secret string, accessDur, refreshDur time.Duration) *TokenManager {
	return &TokenManager{
		secretKey:       []byte(secret),
		accessDuration:  accessDur,
		refreshDuration: refreshDur,
		issuer:          "kfc-crm",
	}
}

// ═══════════════════════════════════════════════════════════════════
// GENERATE
// ═══════════════════════════════════════════════════════════════════

// GenerateAccessToken signs a short-lived token carrying identity.
func (tm *TokenManager) GenerateAccessToken(
	userID uuid.UUID,
	email string,
	role domain.Role,
) (string, error) {
	if userID == uuid.Nil {
		return "", errors.New("user ID is required")
	}
	if email == "" {
		return "", errors.New("email is required")
	}
	if !role.IsValid() {
		return "", errors.New("invalid role")
	}

	now := time.Now()

	claims := Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessDuration)),
			ID:        uuid.NewString(),
		},
	}

	return tm.sign(claims)
}

// GenerateRefreshToken signs a long-lived token carrying only the user ID.
func (tm *TokenManager) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	if userID == uuid.Nil {
		return "", errors.New("user ID is required")
	}

	now := time.Now()

	claims := Claims{
		UserID:    userID,
		TokenType: RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tm.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.refreshDuration)),
			ID:        uuid.NewString(),
		},
	}

	return tm.sign(claims)
}

// ═══════════════════════════════════════════════════════════════════
// VALIDATE
// ═══════════════════════════════════════════════════════════════════

// ValidateAccessToken validates and returns claims for an access token.
func (tm *TokenManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	claims, err := tm.parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != AccessToken {
		return nil, errors.New("not an access token")
	}
	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID.
func (tm *TokenManager) ValidateRefreshToken(tokenString string) (uuid.UUID, error) {
	claims, err := tm.parse(tokenString)
	if err != nil {
		return uuid.Nil, err
	}
	if claims.TokenType != RefreshToken {
		return uuid.Nil, errors.New("not a refresh token")
	}
	return claims.UserID, nil
}

// ═══════════════════════════════════════════════════════════════════
// DURATIONS
// ═══════════════════════════════════════════════════════════════════

func (tm *TokenManager) GetAccessDuration() time.Duration  { return tm.accessDuration }
func (tm *TokenManager) GetRefreshDuration() time.Duration { return tm.refreshDuration }

// ═══════════════════════════════════════════════════════════════════
// INTERNALS
// ═══════════════════════════════════════════════════════════════════

func (tm *TokenManager) sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(tm.secretKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (tm *TokenManager) parse(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return tm.secretKey, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(tm.issuer),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
