package ports

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type EventPublisher interface {
	Publish(ctx context.Context, events ...shared.Event) error
}

type EventHandler interface {
	EventName() string
	Handle(ctx context.Context, e shared.Event) error
}

type EventBus interface {
	EventPublisher
	Subscribe(handler EventHandler)
}
