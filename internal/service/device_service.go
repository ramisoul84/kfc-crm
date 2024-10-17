package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// DeviceService handles device business logic.
type DeviceService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateDeviceRequest) (*domain.DeviceResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.DeviceResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.DeviceFilter) ([]*domain.DeviceResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateDeviceRequest) (*domain.DeviceResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error
}

type deviceService struct {
	deviceRepo repository.DeviceRepository

	rbacService RBACService
	logger      *logger.Logger
}

// NewDeviceService creates a DeviceService.
func NewDeviceService(
	deviceRepo repository.DeviceRepository,
	rbacService RBACService,
	log *logger.Logger,
) DeviceService {
	return &deviceService{
		deviceRepo:  deviceRepo,
		rbacService: rbacService,
		logger:      log,
	}
}

// ═══════════════════════════════════════════════════════════════════
// CREATE
// ═══════════════════════════════════════════════════════════════════

func (s *deviceService) Create(
	ctx context.Context,
	actor *domain.User,
	req *domain.CreateDeviceRequest,
) (*domain.DeviceResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("creating device",
		"actor_id", actor.ID,
		"serial_number", req.SerialNumber,
		"type", req.Type,
		"restaurant_id", req.RestaurantID,
	)

	// 1. Authorization
	if err := s.rbacService.CanCreateDevice(ctx, actor, req.RestaurantID); err != nil {
		log.Warn("create device denied",
			"actor_id", actor.ID,
			"restaurant_id", req.RestaurantID,
			"error", err,
		)
		return nil, err
	}

	// 2. Reject duplicate serial number early for a clearer error
	if _, err := s.deviceRepo.GetBySerialNumber(ctx, req.SerialNumber); err == nil {
		log.Warn("create device: serial exists", "serial_number", req.SerialNumber)
		return nil, domain.NewConflictError("device with this serial number already exists")
	}

	// 3. Build entity
	device := &domain.Device{
		ID:           uuid.New(),
		RestaurantID: req.RestaurantID,
		Type:         req.Type,
		SerialNumber: req.SerialNumber,
		IsActive:     false, // becomes active only after pairing
	}

	// 4. Persist
	if err := s.deviceRepo.Create(ctx, device); err != nil {
		log.Error("create device failed",
			"device_id", device.ID,
			"serial_number", device.SerialNumber,
			"error", err,
		)
		return nil, err
	}

	log.Info("device created",
		"device_id", device.ID,
		"serial_number", device.SerialNumber,
		"type", device.Type,
		"restaurant_id", device.RestaurantID,
		"actor_id", actor.ID,
	)

	return device.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// READ
// ═══════════════════════════════════════════════════════════════════

func (s *deviceService) GetByID(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) (*domain.DeviceResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("getting device", "actor_id", actor.ID, "device_id", id)

	// 1. Load (RBAC scope check needs the device's restaurant)
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get device: load", "device_id", id, "error", err)
		return nil, err
	}

	// 2. Permission + scope
	if err := s.rbacService.CanReadDevice(ctx, actor, device); err != nil {
		log.Warn("get device denied",
			"actor_id", actor.ID,
			"device_id", id,
			"error", err,
		)
		return nil, err
	}

	return device.ToResponse(), nil
}

func (s *deviceService) List(
	ctx context.Context,
	actor *domain.User,
	filter domain.DeviceFilter,
) ([]*domain.DeviceResponse, int, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("listing devices", "actor_id", actor.ID, "role", actor.Role)

	// 1. Permission
	if err := s.rbacService.CheckPermission(actor, domain.PermDeviceRead); err != nil {
		log.Warn("list devices denied", "actor_id", actor.ID, "error", err)
		return nil, 0, err
	}

	// 2. Scope
	switch actor.Role {
	case domain.RoleSuperAdmin:
		// no filter
	case domain.RoleRegionalManager:
		// RM filters by region — the repo can't filter by region directly,
		// so we let the caller pass restaurant_id. For a region-wide list,
		// the caller should iterate over their restaurants. For now, require restaurant_id.
		if filter.RestaurantID == nil {
			return nil, 0, domain.NewValidationError("restaurant_id is required")
		}
	case domain.RoleRestaurantManager, domain.RoleShiftManager:
		if actor.RestaurantID != nil {
			filter.RestaurantID = actor.RestaurantID
		} else {
			return nil, 0, domain.NewAuthorizationError("no restaurant assigned")
		}
	default:
		return nil, 0, domain.NewAuthorizationError("insufficient permissions")
	}

	// 3. Load
	devices, total, err := s.deviceRepo.List(ctx, filter)
	if err != nil {
		log.Error("list devices failed", "error", err)
		return nil, 0, err
	}

	// 4. Convert
	out := make([]*domain.DeviceResponse, 0, len(devices))
	for _, d := range devices {
		out = append(out, d.ToResponse())
	}

	log.Debug("devices listed", "count", len(out))

	return out, total, nil
}

// ═══════════════════════════════════════════════════════════════════
// UPDATE
// ═══════════════════════════════════════════════════════════════════

func (s *deviceService) Update(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
	req *domain.UpdateDeviceRequest,
) (*domain.DeviceResponse, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("updating device", "actor_id", actor.ID, "device_id", id)

	// 1. Load
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("update device: load", "device_id", id, "error", err)
		return nil, err
	}

	// 2. Authorization
	if err := s.rbacService.CanUpdateDevice(ctx, actor, device); err != nil {
		log.Warn("update device denied",
			"actor_id", actor.ID,
			"device_id", id,
			"error", err,
		)
		return nil, err
	}

	// 3. Apply updates
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}

	// 4. Persist
	if err := s.deviceRepo.Update(ctx, device); err != nil {
		log.Error("update device failed", "device_id", id, "error", err)
		return nil, err
	}

	log.Info("device updated",
		"device_id", id,
		"is_active", device.IsActive,
		"actor_id", actor.ID,
	)

	return device.ToResponse(), nil
}

// ═══════════════════════════════════════════════════════════════════
// DELETE
// ═══════════════════════════════════════════════════════════════════

func (s *deviceService) Delete(
	ctx context.Context,
	actor *domain.User,
	id uuid.UUID,
) error {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("deleting device", "actor_id", actor.ID, "device_id", id)

	// 1. Load
	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("delete device: load", "device_id", id, "error", err)
		return err
	}

	// 2. Authorization
	if err := s.rbacService.CanDeleteDevice(ctx, actor, device); err != nil {
		log.Warn("delete device denied",
			"actor_id", actor.ID,
			"device_id", id,
			"error", err,
		)
		return err
	}

	// 3. Delete
	if err := s.deviceRepo.Delete(ctx, id); err != nil {
		log.Error("delete device failed", "device_id", id, "error", err)
		return err
	}

	log.Info("device deleted", "device_id", id, "actor_id", actor.ID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════

// generatePairingCode returns a random 6-digit code.
func generatePairingCode() (string, error) {
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate pairing code: %w", err)
		}
		code[i] = byte('0' + n.Int64())
	}
	return string(code), nil
}
