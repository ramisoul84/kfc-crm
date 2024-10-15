package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
	"github.com/ramisoul84/kfc-crm/pkg/password"
)

// UserService handles user management.
type UserService interface {
	CreateUser(ctx context.Context, actor *domain.User, req *domain.CreateUserRequest) (*domain.UserResponse, error)
	FirstLoginSetup(ctx context.Context, userID uuid.UUID, req *domain.FirstLoginSetupRequest) error
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.UserResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.UserFilter) ([]*domain.UserResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateUserRequest) (*domain.UserResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error
}

type userService struct {
	userRepo    repository.UserRepository
	rbacService RBACService
	logger      *logger.Logger
}

// NewUserService creates a UserService.
func NewUserService(
	userRepo repository.UserRepository,
	rbacService RBACService,
	log *logger.Logger,
) UserService {
	return &userService{
		userRepo:    userRepo,
		rbacService: rbacService,
		logger:      log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *userService) CreateUser(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateUserRequest,
) (*domain.UserResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating user",
		"actor_id", actor.ID,
		"email", req.Email,
		"role", req.Role,
	)

	// 1. Authorization — permission, hierarchy, existence, scope
	if err := s.rbacService.CanCreateUser(ctx, actor, req); err != nil {
		log.Warn("create user denied",
			"actor_id", actor.ID,
			"actor_role", actor.Role,
			"email", req.Email,
			"role", req.Role,
			"error", err,
		)
		return nil, err
	}

	// 2. Reject duplicate email early
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		log.Error("create user: check email", "error", err)
		return nil, err
	}
	if exists {
		log.Warn("create user: email exists", "email", req.Email)
		return nil, domain.NewConflictError("user with this email already exists")
	}

	// 3. Generate and hash the initial password
	initialPassword, err := password.GenerateRandomPassword(16)
	if err != nil {
		log.Error("create user: generate password", "error", err)
		return nil, domain.NewInternalError(err)
	}

	hashed, err := password.HashPassword(initialPassword)
	if err != nil {
		log.Error("create user: hash password", "error", err)
		return nil, domain.NewInternalError(err)
	}

	// 4. Build entity
	user := &domain.User{
		ID:               uuid.New(),
		Email:            req.Email,
		PasswordHash:     hashed,
		Role:             req.Role,
		RegionID:         req.RegionID,
		RestaurantID:     req.RestaurantID,
		IsActive:         true,
		ProfileCompleted: false,
	}

	// 5. Persist
	if err := s.userRepo.Create(ctx, user); err != nil {
		log.Error("create user failed",
			"user_id", user.ID,
			"email", user.Email,
			"error", err,
		)
		return nil, err
	}

	log.Info("user created",
		"user_id", user.ID,
		"email", user.Email,
		"role", user.Role,
		"actor_id", actor.ID,
	)

	// TODO: send welcome email with initialPassword via notification service

	return user.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// FIRST LOGIN SETUP
// ═══════════════════════════════════════════════════════════════════

func (s *userService) FirstLoginSetup(
	ctx context.Context,
	userID uuid.UUID,
	req *domain.FirstLoginSetupRequest,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("first login setup", "user_id", userID)

	// 1. Load user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Error("first login setup: load user", "user_id", userID, "error", err)
		return err
	}

	// 2. Must not be already completed
	if user.ProfileCompleted {
		log.Warn("first login setup: already completed", "user_id", userID)
		return domain.NewConflictError("profile already completed")
	}

	// 3. Hash new password
	hashed, err := password.HashPassword(req.NewPassword)
	if err != nil {
		log.Error("first login setup: hash password", "user_id", userID, "error", err)
		return domain.NewInternalError(err)
	}

	// 4. Persist profile + password atomically
	if err := s.userRepo.CompleteProfile(
		ctx, userID, req.FirstName, req.LastName, req.Phone, hashed,
	); err != nil {
		log.Error("first login setup: save", "user_id", userID, "error", err)
		return err
	}

	log.Info("profile completed", "user_id", userID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *userService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.UserResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("getting user", "actor_id", actor.ID, "target_id", id)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermUserRead); err != nil {
		log.Warn("get user denied: permission", "actor_id", actor.ID, "error", err)
		return nil, err
	}

	// 2. Load
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get user failed", "target_id", id, "error", err)
		return nil, err
	}

	// 3. Scope
	if err := s.rbacService.CanReadUser(actor, user); err != nil {
		log.Warn("get user denied: scope",
			"actor_id", actor.ID,
			"target_id", id,
			"error", err,
		)
		return nil, err
	}

	return user.ToResponse(), nil
}

func (s *userService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.UserFilter,
) ([]*domain.UserResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("listing users", "actor_id", actor.ID, "role", actor.Role)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermUserRead); err != nil {
		log.Warn("list users denied", "actor_id", actor.ID, "error", err)
		return nil, 0, err
	}

	// 2. Scope
	switch actor.Role {
	case domain.RoleSuperAdmin:
		// no filter
	case domain.RoleRegionalManager:
		if actor.RegionID != nil {
			filter.RegionID = actor.RegionID
		}
	case domain.RoleRestaurantManager, domain.RoleShiftManager:
		if actor.RestaurantID != nil {
			filter.RestaurantID = actor.RestaurantID
		}
	default:
		return nil, 0, domain.NewAuthorizationError("insufficient permissions")
	}

	// 3. Load
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		log.Error("list users failed", "error", err)
		return nil, 0, err
	}

	// 4. Convert
	out := make([]*domain.UserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, u.ToResponse())
	}

	log.Debug("users listed", "count", len(out))

	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *userService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateUserRequest,
) (*domain.UserResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("updating user", "actor_id", actor.ID, "target_id", id)

	// 1. Load target
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("update user: load target", "target_id", id, "error", err)
		return nil, err
	}

	// 2. Authorization — permission, self-deactivation, hierarchy, scope
	if err := s.rbacService.CanUpdateUser(actor, target, req); err != nil {
		log.Warn("update user denied",
			"actor_id", actor.ID,
			"target_id", id,
			"error", err,
		)
		return nil, err
	}

	// 3. Apply updates
	if req.FirstName != nil {
		target.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		target.LastName = *req.LastName
	}
	if req.Phone != nil {
		target.Phone = *req.Phone
	}
	if req.IsActive != nil {
		target.IsActive = *req.IsActive
	}

	// 4. Persist
	if err := s.userRepo.Update(ctx, target); err != nil {
		log.Error("update user failed", "target_id", id, "error", err)
		return nil, err
	}

	log.Info("user updated",
		"target_id", id,
		"actor_id", actor.ID,
	)

	return target.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *userService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("deleting user", "actor_id", actor.ID, "target_id", id)

	// 1. Load target
	target, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("delete user: load target", "target_id", id, "error", err)
		return err
	}

	// 2. Authorization — permission, self-deletion, hierarchy, scope
	if err := s.rbacService.CanDeleteUser(actor, target); err != nil {
		log.Warn("delete user denied",
			"actor_id", actor.ID,
			"target_id", id,
			"error", err,
		)
		return err
	}

	// 3. Delete (soft)
	if err := s.userRepo.Delete(ctx, id); err != nil {
		log.Error("delete user failed", "target_id", id, "error", err)
		return err
	}

	log.Info("user deleted", "target_id", id, "actor_id", actor.ID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// CHANGE PASSWORD
// ═══════════════════════════════════════════════════════════════════

func (s *userService) ChangePassword(
	ctx context.Context,
	userID uuid.UUID,
	oldPassword, newPassword string,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("changing password", "user_id", userID)

	// 1. Load user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Error("change password: load user", "user_id", userID, "error", err)
		return err
	}

	// 2. Verify old password
	if err := password.CheckPassword(oldPassword, user.PasswordHash); err != nil {
		log.Warn("change password: invalid old password", "user_id", userID)
		return domain.NewAuthenticationError("invalid current password")
	}

	// 3. Reject same password
	if oldPassword == newPassword {
		return domain.NewValidationError("new password must be different from current password")
	}

	// 4. Hash new password
	hashed, err := password.HashPassword(newPassword)
	if err != nil {
		log.Error("change password: hash", "user_id", userID, "error", err)
		return domain.NewInternalError(err)
	}

	// 5. Persist
	if err := s.userRepo.UpdatePassword(ctx, userID, hashed); err != nil {
		log.Error("change password: save", "user_id", userID, "error", err)
		return err
	}

	log.Info("password changed", "user_id", userID)

	return nil
}
