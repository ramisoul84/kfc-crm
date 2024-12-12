package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/ramisoul84/kfc-crm/internal/client"
	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/ctxutil"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

const pairingCodeTTL = 15 * time.Minute

// DeviceService handles device business logic.
type DeviceService interface {
	Create(ctx context.Context, actor *domain.User, req *domain.CreateDeviceRequest) (*domain.DeviceResponse, error)
	GetByID(ctx context.Context, actor *domain.User, id uuid.UUID) (*domain.DeviceResponse, error)
	List(ctx context.Context, actor *domain.User, filter domain.DeviceFilter) ([]*domain.DeviceResponse, int, error)
	Update(ctx context.Context, actor *domain.User, id uuid.UUID, req *domain.UpdateDeviceRequest) (*domain.DeviceResponse, error)
	Delete(ctx context.Context, actor *domain.User, id uuid.UUID) error

	PairAllDevicesInRestaurant(ctx context.Context, actor *domain.User, restaurantID uuid.UUID) (*domain.PairDevicesResult, error)
	PairDevice(ctx context.Context, actor *domain.User, deviceID uuid.UUID) (*domain.PairDeviceResult, error)
}

type deviceService struct {
	deviceRepo         repository.DeviceRepository
	restaurantRepo     repository.RestaurantRepository
	rbacService        RBACService
	notificationClient client.NotificationClient
	userAPIClient      client.UserAPIClient
	logger             *logger.Logger
}

func NewDeviceService(
	deviceRepo repository.DeviceRepository,
	restaurantRepo repository.RestaurantRepository,
	rbacService RBACService,
	notificationClient client.NotificationClient,
	userAPIClient client.UserAPIClient,
	log *logger.Logger,
) DeviceService {
	return &deviceService{
		deviceRepo:         deviceRepo,
		restaurantRepo:     restaurantRepo,
		rbacService:        rbacService,
		notificationClient: notificationClient,
		userAPIClient:      userAPIClient,
		logger:             log,
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

	// 2. Reject duplicate serial number early for a clearer error.
	// Distinguish "not found" (proceed) from real DB errors (fail).
	existing, err := s.deviceRepo.GetBySerialNumber(ctx, req.SerialNumber)
	if err != nil && !domain.IsNotFoundError(err) {
		log.Error("create device: serial lookup failed",
			"serial_number", req.SerialNumber,
			"error", err,
		)
		return nil, err
	}
	if existing != nil {
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

	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("get device: load", "device_id", id, "error", err)
		return nil, err
	}

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
		// TODO: support region-wide listing. For now, require restaurant_id
		// because the repository can only filter by restaurant.
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

	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("update device: load", "device_id", id, "error", err)
		return nil, err
	}

	if err := s.rbacService.CanUpdateDevice(ctx, actor, device); err != nil {
		log.Warn("update device denied",
			"actor_id", actor.ID,
			"device_id", id,
			"error", err,
		)
		return nil, err
	}

	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}

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

	device, err := s.deviceRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("delete device: load", "device_id", id, "error", err)
		return err
	}

	if err := s.rbacService.CanDeleteDevice(ctx, actor, device); err != nil {
		log.Warn("delete device denied",
			"actor_id", actor.ID,
			"device_id", id,
			"error", err,
		)
		return err
	}

	// Revoke any cached pairing code in userapi before deleting.
	// Best-effort: the device row is gone either way.
	if err := s.userAPIClient.RevokePairingCode(ctx, device.SerialNumber); err != nil {
		log.Warn("failed to revoke pairing code in userapi",
			"serial_number", device.SerialNumber,
			"error", err,
		)
	}

	// Revoke all issued tokens for this device. Best-effort: if userapi
	// is unreachable, the tokens still expire on their own.
	if err := s.userAPIClient.RevokeDevice(ctx, device.ID.String()); err != nil {
		log.Warn("failed to revoke device in userapi",
			"device_id", device.ID,
			"error", err,
		)
	}

	if err := s.deviceRepo.Delete(ctx, id); err != nil {
		log.Error("delete device failed", "device_id", id, "error", err)
		return err
	}

	log.Info("device deleted", "device_id", id, "actor_id", actor.ID)

	return nil
}

// ═══════════════════════════════════════════════════════════════════
// PAIR ALL DEVICES IN RESTAURANT
// ═══════════════════════════════════════════════════════════════════

// PairAllDevicesInRestaurant generates one pairing code per device in the
// restaurant and dispatches them to userapi (the only store) and to the
// manager by email.
//
// Design notes:
//   - CRM does NOT persist pairing codes. userapi's Redis is the only store.
//   - userapi must succeed: if it doesn't, the codes are useless, so the
//     operation fails and no email is sent. This avoids emailing codes
//     that will never work.
//   - Individual code-generation failures do not abort the batch; the
//     result reports per-device success/error.
//   - "Pair all" supersedes any previous batch. userapi replaces the
//     restaurant's codes atomically (see the ClearRestaurant variant of
//     CachePairingCodes on the userapi side).
//   - All registered devices are paired regardless of IsActive, because
//     pairing is what activates a device.
func (s *deviceService) PairAllDevicesInRestaurant(
	ctx context.Context,
	actor *domain.User,
	restaurantID uuid.UUID,
) (*domain.PairDevicesResult, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("pairing all devices",
		"actor_id", actor.ID,
		"restaurant_id", restaurantID,
	)

	// 1. Authorization
	if err := s.rbacService.CanPairDevices(ctx, actor, restaurantID); err != nil {
		log.Warn("pair all devices denied",
			"actor_id", actor.ID,
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, err
	}

	// 2. Load restaurant
	restaurant, err := s.restaurantRepo.GetByID(ctx, restaurantID)
	if err != nil {
		log.Error("pair all devices: load restaurant",
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, err
	}

	// 3. Load devices
	devices, err := s.deviceRepo.ListByRestaurant(ctx, restaurantID)
	if err != nil {
		log.Error("pair all devices: load devices",
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, err
	}
	if len(devices) == 0 {
		return nil, domain.NewValidationError("no devices registered at this restaurant")
	}

	// 4. Generate one code per device. Failures are recorded per-device.
	expiresAt := time.Now().Add(pairingCodeTTL)
	entries := make([]domain.PairingEntry, 0, len(devices))

	for _, d := range devices {
		entry := domain.PairingEntry{
			DeviceID:     d.ID,
			SerialNumber: d.SerialNumber,
			DeviceType:   string(d.Type),
		}

		code, err := generatePairingCode()
		if err != nil {
			entry.Error = "code generation failed"
			entries = append(entries, entry)
			log.Error("pair all: code generation failed",
				"device_id", d.ID,
				"error", err,
			)
			continue
		}

		entry.Code = code
		entries = append(entries, entry)
	}

	// 5. Collect successes. If none, fail before touching userapi.
	successes := successfulEntries(entries)
	if len(successes) == 0 {
		log.Error("pair all: no codes generated",
			"restaurant_id", restaurantID,
			"device_count", len(devices),
		)
		return nil, domain.NewInternalError(
			errors.New("failed to generate any pairing codes"),
		)
	}

	// 6. Send codes to userapi — the ONLY store. Failure is fatal.
	if err := s.cacheCodesInUserAPI(ctx, restaurantID, expiresAt, successes); err != nil {
		log.Error("pair all: userapi cache failed",
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, domain.NewInternalError(
			fmt.Errorf("pairing service unavailable: %w", err),
		)
	}

	// 7. Email the manager. Only reached once codes are confirmed stored.
	// Failure is non-fatal: the manager can see the codes in the UI.
	if err := s.sendPairingEmail(ctx, actor.Email, restaurantID, restaurant.Name, expiresAt, successes); err != nil {
		log.Error("pair all: pairing email failed",
			"restaurant_id", restaurantID,
			"error", err,
		)
	}

	log.Info("pair all devices completed",
		"actor_id", actor.ID,
		"restaurant_id", restaurantID,
		"requested", len(devices),
		"succeeded", len(successes),
		"failed", len(entries)-len(successes),
		"expires_at", expiresAt,
	)

	return &domain.PairDevicesResult{
		RestaurantID: restaurantID,
		ExpiresAt:    expiresAt,
		Devices:      entries,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════
// PAIR A SINGLE DEVICE
// ═══════════════════════════════════════════════════════════════════

// PairDevice generates a fresh pairing code for a single device.
// Used when only one device needs to be re-paired (e.g. a replacement
// unit, or a code that expired before the manager reached the device).
func (s *deviceService) PairDevice(
	ctx context.Context,
	actor *domain.User,
	deviceID uuid.UUID,
) (*domain.PairDeviceResult, error) {
	log := s.logger.WithRequestID(ctxutil.GetRequestID(ctx))

	log.Debug("pairing single device",
		"actor_id", actor.ID,
		"device_id", deviceID,
	)

	// 1. Load device
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		log.Error("pair device: load", "device_id", deviceID, "error", err)
		return nil, err
	}

	// 2. Authorization
	if err := s.rbacService.CanPairDevice(ctx, actor, device); err != nil {
		log.Warn("pair device denied",
			"actor_id", actor.ID,
			"device_id", deviceID,
			"error", err,
		)
		return nil, err
	}

	// 3. Revoke any previous code for this serial. Best-effort — if
	// userapi has no code cached, the revoke is a no-op. If revoke
	// fails, the new code will still supersede the old one when
	// CachePairingCode overwrites the key.
	if err := s.userAPIClient.RevokePairingCode(ctx, device.SerialNumber); err != nil {
		log.Warn("pair device: failed to revoke old code in userapi",
			"serial_number", device.SerialNumber,
			"error", err,
		)
		// Continue: the subsequent CachePairingCode overwrites the key.
	}

	// 4. Generate new code
	code, err := generatePairingCode()
	if err != nil {
		return nil, domain.NewInternalError(err)
	}

	expiresAt := time.Now().Add(pairingCodeTTL)

	// 5. Cache in userapi — the ONLY store. Failure is fatal.
	if err := s.userAPIClient.CachePairingCode(ctx, &client.CachePairingCodeRequest{
		RestaurantID: device.RestaurantID.String(),
		SerialNumber: device.SerialNumber,
		DeviceID:     device.ID.String(),
		Code:         code,
		ExpiresAt:    expiresAt,
	}); err != nil {
		log.Error("pair device: userapi cache failed",
			"device_id", deviceID,
			"error", err,
		)
		return nil, domain.NewInternalError(
			fmt.Errorf("pairing service unavailable: %w", err),
		)
	}

	log.Info("single device paired",
		"actor_id", actor.ID,
		"device_id", deviceID,
		"serial_number", device.SerialNumber,
		"expires_at", expiresAt,
	)

	return &domain.PairDeviceResult{
		DeviceID:     device.ID,
		SerialNumber: device.SerialNumber,
		DeviceType:   string(device.Type),
		Code:         code,
		ExpiresAt:    expiresAt,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════════════

// successfulEntries filters entries that have a code (no error).
func successfulEntries(entries []domain.PairingEntry) []domain.PairingEntry {
	out := make([]domain.PairingEntry, 0, len(entries))
	for _, e := range entries {
		if e.Code != "" {
			out = append(out, e)
		}
	}
	return out
}

// cacheCodesInUserAPI sends the successfully generated pairing codes to
// userapi for Redis storage. This is the only place codes are persisted.
func (s *deviceService) cacheCodesInUserAPI(
	ctx context.Context,
	restaurantID uuid.UUID,
	expiresAt time.Time,
	entries []domain.PairingEntry,
) error {
	if len(entries) == 0 {
		return nil
	}

	req := &client.CachePairingCodesRequest{
		RestaurantID: restaurantID.String(),
		ExpiresAt:    expiresAt,
		Codes:        make([]client.PairingCodeEntry, 0, len(entries)),
	}
	for _, e := range entries {
		req.Codes = append(req.Codes, client.PairingCodeEntry{
			SerialNumber: e.SerialNumber,
			DeviceID:     e.DeviceID.String(),
			Code:         e.Code,
		})
	}

	return s.userAPIClient.CachePairingCodes(ctx, req)
}

// sendPairingEmail emails the manager with the successfully generated
// pairing codes. Best-effort — errors are returned for logging.
func (s *deviceService) sendPairingEmail(
	ctx context.Context,
	to string,
	restaurantID uuid.UUID,
	restaurantName string,
	expiresAt time.Time,
	entries []domain.PairingEntry,
) error {
	if len(entries) == 0 {
		return nil
	}

	req := &client.DevicePairingEmail{
		To:             to,
		RestaurantID:   restaurantID.String(),
		RestaurantName: restaurantName,
		ExpiresAt:      expiresAt,
		Devices:        make([]client.DevicePairingItem, 0, len(entries)),
	}
	for _, e := range entries {
		req.Devices = append(req.Devices, client.DevicePairingItem{
			SerialNumber: e.SerialNumber,
			DeviceType:   e.DeviceType,
			Code:         e.Code,
		})
	}

	return s.notificationClient.SendDevicePairingEmail(ctx, req)
}

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
