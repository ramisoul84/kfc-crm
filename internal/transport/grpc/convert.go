package grpc

import (
	menuv1 "github.com/ramisoul84/kfc-crm/gen/menu/v1"
	"github.com/ramisoul84/kfc-crm/internal/domain"
)

// toProtoMenu converts a domain EffectiveMenu to its proto representation.
func toProtoMenu(m *domain.EffectiveMenu) *menuv1.GetEffectiveMenuResponse {
	if m == nil {
		return nil
	}

	cats := make([]*menuv1.Category, 0, len(m.Categories))
	for _, c := range m.Categories {
		cats = append(cats, toProtoCategory(c))
	}

	return &menuv1.GetEffectiveMenuResponse{
		RestaurantId:  m.RestaurantID.String(),
		GeneratedAt:   m.GeneratedAt.Unix(),
		Currency:      "RUB",
		Categories:    cats,
		SchemaVersion: 1,
	}
}

func toProtoCategory(c *domain.EffectiveCategory) *menuv1.Category {
	if c == nil {
		return nil
	}

	items := make([]*menuv1.Item, 0, len(c.Items))
	for _, it := range c.Items {
		items = append(items, toProtoItem(it))
	}

	return &menuv1.Category{
		Id:        c.ID.String(),
		Name:      c.Name,
		Code:      c.Code,
		ImageUrl:  c.ImageURL,
		SortOrder: int32(c.SortOrder),
		Items:     items,
	}
}

func toProtoItem(it *domain.EffectiveMenuItem) *menuv1.Item {
	if it == nil {
		return nil
	}

	vars := make([]*menuv1.Variation, 0, len(it.Variations))
	for _, v := range it.Variations {
		vars = append(vars, toProtoVariation(v))
	}

	return &menuv1.Item{
		Id:          it.ID.String(),
		ProductCode: it.ProductCode,
		Name:        it.Name,
		Description: it.Description,
		ImageUrl:    it.ImageURL,
		BasePrice:   it.BasePrice,
		FinalPrice:  it.FinalPrice,
		Currency:    it.Currency,
		IsAvailable: it.IsAvailable,
		SortOrder:   int32(it.SortOrder),
		Promotion:   toProtoPromotion(it.Promotion),
		Variations:  vars,
	}
}

func toProtoPromotion(p *domain.EffectivePromotion) *menuv1.Promotion {
	if p == nil {
		return nil
	}

	return &menuv1.Promotion{
		Id:              p.ID.String(),
		Name:            p.Name,
		DiscountType:    string(p.DiscountType),
		DiscountValue:   p.DiscountValue,
		OriginalPrice:   p.OriginalPrice,
		DiscountedPrice: p.DiscountedPrice,
		EndsAt:          p.EndsAt.Unix(),
	}
}

func toProtoVariation(v *domain.EffectiveVariation) *menuv1.Variation {
	if v == nil {
		return nil
	}

	return &menuv1.Variation{
		Id:         v.ID.String(),
		Name:       v.Name,
		PriceDelta: v.PriceDelta,
		FinalPrice: v.FinalPrice,
		IsDefault:  v.IsDefault,
		SortOrder:  int32(v.SortOrder),
	}
}
