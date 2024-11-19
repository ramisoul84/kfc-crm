package events

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/ramisoul84/kfc-crm/pkg/logger"
	"github.com/ramisoul84/kfc-crm/pkg/outbox"
)

// Publisher records domain events in the outbox; pkg/outbox's relay
// delivers them to Kafka. Writing to Postgres instead of Kafka directly
// means a Kafka outage delays menu updates instead of losing them.
type Publisher struct {
	db     *sqlx.DB
	notify func()
	logger *logger.Logger
}

// NewPublisher builds a publisher. notify wakes the relay; may be nil.
func NewPublisher(db *sqlx.DB, notify func(), log *logger.Logger) *Publisher {
	if notify == nil {
		notify = func() {}
	}
	return &Publisher{db: db, notify: notify, logger: log}
}

// PublishMenuUpdated records a menu-change event. Call it after any
// successful mutation to a menu entity.
//
// Errors are logged, not returned: the menu change itself is already
// committed. userapi's cache also expires on its own (bounded TTL and
// promotion boundaries), so a missed event means a short delay at worst.
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
	if p == nil {
		return
	}

	evt := MenuUpdatedEvent{
		RestaurantID: restaurantID,
		RegionID:     regionID,
		Reason:       reason,
		OccurredAt:   time.Now().UTC(),
	}

	// Detached from the request: a client disconnecting right after the
	// change must not drop its event.
	ctx = context.WithoutCancel(ctx)
	if err := outbox.Insert(ctx, p.db, TopicMenuUpdated, restaurantID, evt); err != nil {
		p.logger.Error("failed to record menu update event",
			"restaurant_id", restaurantID,
			"reason", reason,
			"error", err,
		)
		return
	}
	p.notify()

	p.logger.Info("menu update event recorded",
		"restaurant_id", restaurantID,
		"reason", reason,
	)
}
