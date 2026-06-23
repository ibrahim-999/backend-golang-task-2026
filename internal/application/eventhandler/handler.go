package eventhandler

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type funcHandler struct {
	name string
	fn   func(ctx context.Context, e shared.Event) error
}

func New(name string, fn func(ctx context.Context, e shared.Event) error) ports.EventHandler {
	return funcHandler{name: name, fn: fn}
}

func (h funcHandler) EventName() string { return h.name }

func (h funcHandler) Handle(ctx context.Context, e shared.Event) error {
	return h.fn(ctx, e)
}
