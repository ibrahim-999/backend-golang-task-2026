package eventhandler

import (
	"context"
	"strings"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/audit"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

func AuditHandler(audits ports.AuditRepository) ports.EventHandler {
	return New("*", func(ctx context.Context, e shared.Event) error {
		entity := entityOf(e.EventName())
		entry, err := audit.NewEntry(nil, e.EventName(), entity, e.AggregateID(), "", "", "")
		if err != nil {
			return err
		}
		return audits.Append(ctx, entry)
	})
}

func entityOf(eventName string) string {
	if i := strings.IndexByte(eventName, '.'); i > 0 {
		return eventName[:i]
	}
	return eventName
}
