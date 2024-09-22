package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	menuv1 "github.com/ramisoul84/kfc-crm/gen/menu/v1"
	"github.com/ramisoul84/kfc-crm/internal/service"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// MenuServer implements menuv1.MenuServiceServer.
type MenuServer struct {
	menuv1.UnimplementedMenuServiceServer

	svc    service.EffectiveMenuService
	logger *logger.Logger
}

func NewMenuServer(svc service.EffectiveMenuService, log *logger.Logger) *MenuServer {
	return &MenuServer{svc: svc, logger: log}
}

func (s *MenuServer) GetEffectiveMenu(
	ctx context.Context,
	req *menuv1.GetEffectiveMenuRequest,
) (*menuv1.GetEffectiveMenuResponse, error) {
	restaurantID, err := uuid.Parse(req.GetRestaurantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid restaurant_id")
	}

	menu, err := s.svc.GetEffectiveMenuInternal(ctx, restaurantID)
	if err != nil {
		s.logger.Error("grpc GetEffectiveMenu failed",
			"restaurant_id", restaurantID,
			"error", err,
		)
		return nil, mapErrorToGRPC(err)
	}

	return toProtoMenu(menu), nil
}
