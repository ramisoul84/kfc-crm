package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	restaurantv1 "github.com/ramisoul84/kfc-crm/gen/restaurant/v1"
	"github.com/ramisoul84/kfc-crm/internal/domain"
	"github.com/ramisoul84/kfc-crm/internal/repository"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// directoryPageSize matches the repositories' maximum page size. The
// directory pages through everything; a chain has a few hundred
// restaurants at most.
const directoryPageSize = 100

// RestaurantServer implements restaurantv1.RestaurantDirectoryServer.
//
// It reads repositories directly rather than going through the
// actor-scoped services: callers are trusted internal services asking
// for public data, so there is no user to authorize.
type RestaurantServer struct {
	restaurantv1.UnimplementedRestaurantDirectoryServer

	restaurants repository.RestaurantRepository
	regions     repository.RegionRepository
	logger      *logger.Logger
}

func NewRestaurantServer(
	restaurants repository.RestaurantRepository,
	regions repository.RegionRepository,
	log *logger.Logger,
) *RestaurantServer {
	return &RestaurantServer{restaurants: restaurants, regions: regions, logger: log}
}

func (s *RestaurantServer) ListRegions(
	ctx context.Context,
	_ *restaurantv1.ListRegionsRequest,
) (*restaurantv1.ListRegionsResponse, error) {
	regions, err := s.activeRegions(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	restaurants, err := s.activeRestaurants(ctx, nil)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}

	counts := make(map[uuid.UUID]int32)
	for _, r := range restaurants {
		counts[r.RegionID]++
	}

	// Only regions that actually have somewhere to order from.
	out := &restaurantv1.ListRegionsResponse{}
	for _, r := range regions {
		if counts[r.ID] == 0 {
			continue
		}
		out.Regions = append(out.Regions, &restaurantv1.Region{
			Id:              r.ID.String(),
			Name:            r.Name,
			Code:            r.Code,
			Timezone:        r.Timezone,
			RestaurantCount: counts[r.ID],
		})
	}
	return out, nil
}

func (s *RestaurantServer) ListRestaurants(
	ctx context.Context,
	req *restaurantv1.ListRestaurantsRequest,
) (*restaurantv1.ListRestaurantsResponse, error) {
	var regionID *uuid.UUID
	if req.GetRegionId() != "" {
		id, err := uuid.Parse(req.GetRegionId())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid region_id")
		}
		regionID = &id
	}

	restaurants, err := s.activeRestaurants(ctx, regionID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	regions, err := s.activeRegions(ctx)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	byID := make(map[uuid.UUID]*domain.Region, len(regions))
	for _, r := range regions {
		byID[r.ID] = r
	}

	out := &restaurantv1.ListRestaurantsResponse{}
	for _, r := range restaurants {
		region, ok := byID[r.RegionID]
		if !ok {
			continue // region deactivated: hide its restaurants too
		}
		out.Restaurants = append(out.Restaurants, toProtoRestaurant(r, region))
	}
	return out, nil
}

func (s *RestaurantServer) GetRestaurant(
	ctx context.Context,
	req *restaurantv1.GetRestaurantRequest,
) (*restaurantv1.Restaurant, error) {
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	r, err := s.restaurants.GetByID(ctx, id)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	region, err := s.regions.GetByID(ctx, r.RegionID)
	if err != nil {
		return nil, mapErrorToGRPC(err)
	}
	if !r.IsActive || !region.IsActive {
		return nil, status.Error(codes.NotFound, "restaurant not found")
	}
	return toProtoRestaurant(r, region), nil
}

func (s *RestaurantServer) activeRegions(ctx context.Context) ([]*domain.Region, error) {
	active := true
	var all []*domain.Region
	for offset := 0; ; offset += directoryPageSize {
		page, total, err := s.regions.List(ctx, domain.RegionFilter{
			IsActive: &active,
			Limit:    directoryPageSize,
			Offset:   offset,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || len(all) >= total {
			return all, nil
		}
	}
}

func (s *RestaurantServer) activeRestaurants(ctx context.Context, regionID *uuid.UUID) ([]*domain.Restaurant, error) {
	active := true
	var all []*domain.Restaurant
	for offset := 0; ; offset += directoryPageSize {
		page, total, err := s.restaurants.List(ctx, domain.RestaurantFilter{
			RegionID: regionID,
			IsActive: &active,
			Limit:    directoryPageSize,
			Offset:   offset,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) == 0 || len(all) >= total {
			return all, nil
		}
	}
}

func toProtoRestaurant(r *domain.Restaurant, region *domain.Region) *restaurantv1.Restaurant {
	return &restaurantv1.Restaurant{
		Id:         r.ID.String(),
		Name:       r.Name,
		Code:       r.Code,
		RegionId:   r.RegionID.String(),
		RegionName: region.Name,
		Address:    r.Address,
		City:       r.City,
		Phone:      r.Phone,
		Timezone:   region.Timezone,
	}
}
