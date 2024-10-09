package events

import (
	"context"
	"time"

	"github.com/ramisoul84/kfc-crm/pkg/kafka"
	"github.com/ramisoul84/kfc-crm/pkg/logger"
)

// Publisher publishes domain events to Kafka.
type Publisher struct {
	producer *kafka.Producer
	logger   *logger.Logger
}

func NewPublisher(producer *kafka.Producer, log *logger.Logger) *Publisher {
	return &Publisher{producer: producer, logger: log}
}

// PublishMenuUpdated emits a menu-change event. Call this after any
// successful mutation to a menu entity.
//
// The event is best-effort: errors are logged but not returned, because
// the DB write already succeeded and rolling it back just because Kafka
// was briefly unavailable would be worse. The 24h TTL on userapi's menu
// cache is the safety net.
//
// restaurantID = "" means "all restaurants" — used for global menu
// changes (categories, items, variations). Overrides and restaurant-
// scoped promotions pass their specific restaurant_id.
func (p *Publisher) PublishMenuUpdated(
	ctx context.Context,
	restaurantID string,
	regionID string,
	reason string,
) {
	if p == nil || p.producer == nil {
		return
	}

	evt := MenuUpdatedEvent{
		RestaurantID: restaurantID,
		RegionID:     regionID,
		Reason:       reason,
		OccurredAt:   time.Now().UTC(),
	}

	// Partition key. Empty restaurantID becomes "all" inside Publish.
	key := restaurantID

	if err := p.producer.Publish(ctx, TopicMenuUpdated, key, evt); err != nil {
		p.logger.Error("failed to publish menu update event",
			"restaurant_id", restaurantID,
			"reason", reason,
			"error", err,
		)
		return
	}

	p.logger.Info("menu update event published",
		"restaurant_id", restaurantID,
		"reason", reason,
	)
}
