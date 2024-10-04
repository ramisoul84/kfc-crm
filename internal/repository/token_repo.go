package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TokenRepository stores refresh tokens and access-token blacklist entries.
type TokenRepository interface {
	// StoreRefreshToken persists a refresh token until ttl expires.
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error

	// IsRefreshTokenValid reports whether the refresh token is still valid.
	IsRefreshTokenValid(ctx context.Context, userID uuid.UUID, token string) (bool, error)

	// RevokeRefreshToken removes a single refresh token.
	RevokeRefreshToken(ctx context.Context, userID uuid.UUID, token string) error

	// RevokeAllRefreshTokens removes all refresh tokens for a user.
	RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error

	// BlacklistAccessToken marks an access token as revoked until it expires.
	BlacklistAccessToken(ctx context.Context, token string, ttl time.Duration) error

	// IsAccessTokenBlacklisted reports whether the access token is revoked.
	IsAccessTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

// Redis key prefixes
const (
	refreshTokenPrefix = "refresh_token"
	blacklistPrefix    = "blacklist"
	userTokensPrefix   = "user_tokens"
)

type tokenRepo struct {
	redis *redis.Client
}

// NewTokenRepository creates a TokenRepository backed by Redis.
func NewTokenRepository(client *redis.Client) TokenRepository {
	return &tokenRepo{redis: client}
}

// ═══════════════════════════════════════════════════════════════════
// REFRESH TOKENS
// ═══════════════════════════════════════════════════════════════════

func (r *tokenRepo) StoreRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	token string,
	ttl time.Duration,
) error {
	tokenHash := hashToken(token)

	pipe := r.redis.Pipeline()
	pipe.Set(ctx, refreshKey(userID, tokenHash), "1", ttl)
	pipe.SAdd(ctx, userTokensKey(userID), tokenHash)
	pipe.Expire(ctx, userTokensKey(userID), ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *tokenRepo) IsRefreshTokenValid(
	ctx context.Context,
	userID uuid.UUID,
	token string,
) (bool, error) {
	tokenHash := hashToken(token)

	exists, err := r.redis.Exists(ctx, refreshKey(userID, tokenHash)).Result()
	if err != nil {
		return false, fmt.Errorf("check refresh token: %w", err)
	}
	return exists > 0, nil
}

func (r *tokenRepo) RevokeRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	token string,
) error {
	tokenHash := hashToken(token)

	pipe := r.redis.Pipeline()
	pipe.Del(ctx, refreshKey(userID, tokenHash))
	pipe.SRem(ctx, userTokensKey(userID), tokenHash)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *tokenRepo) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	hashes, err := r.redis.SMembers(ctx, userTokensKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("list user tokens: %w", err)
	}
	if len(hashes) == 0 {
		return nil
	}

	pipe := r.redis.Pipeline()
	for _, h := range hashes {
		pipe.Del(ctx, refreshKey(userID, h))
	}
	pipe.Del(ctx, userTokensKey(userID))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("revoke all tokens: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════
// ACCESS TOKEN BLACKLIST
// ═══════════════════════════════════════════════════════════════════

func (r *tokenRepo) BlacklistAccessToken(
	ctx context.Context,
	token string,
	ttl time.Duration,
) error {
	tokenHash := hashToken(token)

	if err := r.redis.Set(ctx, blacklistKey(tokenHash), "1", ttl).Err(); err != nil {
		return fmt.Errorf("blacklist token: %w", err)
	}
	return nil
}

func (r *tokenRepo) IsAccessTokenBlacklisted(
	ctx context.Context,
	token string,
) (bool, error) {
	tokenHash := hashToken(token)

	exists, err := r.redis.Exists(ctx, blacklistKey(tokenHash)).Result()
	if err != nil {
		return false, fmt.Errorf("check blacklist: %w", err)
	}
	return exists > 0, nil
}

// ═══════════════════════════════════════════════════════════════════
// KEY HELPERS
// ═══════════════════════════════════════════════════════════════════

func refreshKey(userID uuid.UUID, tokenHash string) string {
	return fmt.Sprintf("%s:%s:%s", refreshTokenPrefix, userID, tokenHash)
}

func userTokensKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", userTokensPrefix, userID)
}

func blacklistKey(tokenHash string) string {
	return fmt.Sprintf("%s:%s", blacklistPrefix, tokenHash)
}

// hashToken returns the SHA-256 hex digest of the token.
// We never store raw tokens — only their hashes.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
