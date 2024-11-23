package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/jwt"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
	"github.com/ramisoul84/kfc-crm/pkg/password"
)

// AuthService handles authentication: login, refresh, logout.
type AuthService interface {
	Login(ctx context.Context, req *domain.UserLoginRequest) (*domain.TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error)
	Logout(ctx context.Context, userID uuid.UUID, accessToken string) error
}

type authService struct {
	userRepo     repository.UserRepository
	tokenRepo    repository.TokenRepository
	tokenManager *jwt.TokenManager
	logger       *logger.Logger
}

// NewAuthService creates an AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	tokenManager *jwt.TokenManager,
	log *logger.Logger,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		tokenManager: tokenManager,
		logger:       log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// LOGIN
// ═══════════════════════════════════════════════════════════════════

func (s *authService) Login(
	ctx context.Context,
	req *domain.UserLoginRequest,
) (*domain.TokenPair, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("login attempt", "email", req.Email)

	// 0. Brute-force guard: too many recent failures lock the account
	//    for the rest of the window, whatever the password.
	if n, err := s.tokenRepo.LoginFailures(ctx, req.Email); err == nil && n >= maxLoginFailures {
		log.Warn("login blocked: too many failures", "email", req.Email)
		return nil, domain.NewTooManyRequestsError("too many failed sign-in attempts, try again in 15 minutes")
	}

	// 1. Look up the user
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Do not leak whether the email exists — not in the message, and
		// not in the timing: spend the same bcrypt work as a real check.
		_ = password.CheckPassword(req.Password, dummyPasswordHash)
		s.loginFailed(ctx, req.Email)
		log.Warn("login failed: user not found", "email", req.Email)
		return nil, domain.NewAuthenticationError("invalid email or password")
	}

	// 2. Verify password
	if err := password.CheckPassword(req.Password, user.PasswordHash); err != nil {
		s.loginFailed(ctx, req.Email)
		log.Warn("login failed: invalid password", "user_id", user.ID)
		return nil, domain.NewAuthenticationError("invalid email or password")
	}

	// 3. Check active status — only after the password, so a stranger
	//    can't learn which accounts exist but are disabled.
	if !user.IsActive {
		log.Warn("login failed: inactive user", "user_id", user.ID)
		return nil, domain.NewAuthenticationError("user account is inactive")
	}
	_ = s.tokenRepo.ResetLoginFailures(ctx, req.Email)

	// 4. Generate tokens
	pair, err := s.generateTokenPair(user)
	if err != nil {
		log.Error("login failed: generate tokens", "user_id", user.ID, "error", err)
		return nil, domain.NewInternalError(err)
	}

	// 5. Store refresh token
	if err := s.tokenRepo.StoreRefreshToken(
		ctx, user.ID, pair.RefreshToken, s.tokenManager.GetRefreshDuration(),
	); err != nil {
		log.Error("login failed: store refresh token", "user_id", user.ID, "error", err)
		return nil, err
	}

	// 6. Update last login (best-effort, async)
	go s.updateLastLogin(user.ID)

	log.Info("login successful",
		"user_id", user.ID,
		"email", user.Email,
		"role", user.Role,
	)

	return pair, nil
}

// Brute-force limits for staff sign-in.
const (
	maxLoginFailures   = 10
	loginFailureWindow = 15 * time.Minute
)

// dummyPasswordHash is a bcrypt hash (cost 12) of a random string, used
// to make unknown-email logins cost the same time as real ones.
const dummyPasswordHash = "$2a$12$uEnUVQPniJuclk/Kk975juKY4AtLW5o3XTR0Vm.LPAa/Dk.HOdw7y"

func (s *authService) loginFailed(ctx context.Context, email string) {
	if _, err := s.tokenRepo.RegisterLoginFailure(ctx, email, loginFailureWindow); err != nil {
		s.logger.Warn("could not record failed login", "error", err)
	}
}

// ═══════════════════════════════════════════════════════════════════
// REFRESH
// ═══════════════════════════════════════════════════════════════════

func (s *authService) RefreshToken(
	ctx context.Context,
	refreshToken string,
) (*domain.TokenPair, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("refresh attempt")

	// 1. Validate the refresh token signature
	userID, err := s.tokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		log.Warn("refresh failed: invalid token")
		return nil, domain.NewAuthenticationError("invalid refresh token")
	}

	// 2. Verify it hasn't been revoked
	valid, err := s.tokenRepo.IsRefreshTokenValid(ctx, userID, refreshToken)
	if err != nil {
		log.Error("refresh failed: check token", "user_id", userID, "error", err)
		return nil, err
	}
	if !valid {
		log.Warn("refresh failed: token not in storage", "user_id", userID)
		return nil, domain.NewAuthenticationError("refresh token not found or revoked")
	}

	// 3. Load the user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Error("refresh failed: load user", "user_id", userID, "error", err)
		return nil, domain.NewAuthenticationError("invalid refresh token")
	}
	if !user.IsActive {
		log.Warn("refresh failed: inactive user", "user_id", userID)
		return nil, domain.NewAuthenticationError("user account is inactive")
	}

	// 4. Rotate: generate a new pair
	pair, err := s.generateTokenPair(user)
	if err != nil {
		log.Error("refresh failed: generate tokens", "user_id", userID, "error", err)
		return nil, domain.NewInternalError(err)
	}

	// 5. Revoke the old refresh token
	if err := s.tokenRepo.RevokeRefreshToken(ctx, userID, refreshToken); err != nil {
		log.Error("refresh: revoke old token", "user_id", userID, "error", err)
		// Continue — the token will expire on its own
	}

	// 6. Store the new refresh token
	if err := s.tokenRepo.StoreRefreshToken(
		ctx, userID, pair.RefreshToken, s.tokenManager.GetRefreshDuration(),
	); err != nil {
		log.Error("refresh: store new token", "user_id", userID, "error", err)
		return nil, err
	}

	log.Info("token refreshed", "user_id", userID)

	return pair, nil
}

// ═══════════════════════════════════════════════════════════════════
// LOGOUT
// ═══════════════════════════════════════════════════════════════════

func (s *authService) Logout(
	ctx context.Context,
	userID uuid.UUID,
	accessToken string,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("logout attempt", "user_id", userID)

	// 1. Revoke all refresh tokens for this user
	if err := s.tokenRepo.RevokeAllRefreshTokens(ctx, userID); err != nil {
		log.Error("logout: revoke refresh tokens", "user_id", userID, "error", err)
		return err
	}

	// 2. Blacklist the current access token until it naturally expires
	if accessToken != "" {
		ttl := s.tokenManager.GetAccessDuration()
		if err := s.tokenRepo.BlacklistAccessToken(ctx, accessToken, ttl); err != nil {
			log.Error("logout: blacklist access token", "user_id", userID, "error", err)
			// Continue — refresh tokens are already revoked
		}
	}

	log.Info("logout successful", "user_id", userID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════

func (s *authService) generateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	access, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refresh, err := s.tokenManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		ExpiresIn:        int64(s.tokenManager.GetAccessDuration().Seconds()),
		TokenType:        "Bearer",
		ProfileCompleted: user.ProfileCompleted,
	}, nil
}

// updateLastLogin updates the last_login_at field in the background.
// Failures are logged but do not affect login.
func (s *authService) updateLastLogin(userID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.userRepo.UpdateLastLogin(ctx, userID); err != nil {
		s.logger.Error("update last login failed", "user_id", userID, "error", err)
	}
}
