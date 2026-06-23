package eventhandler

import (
	"context"
	"encoding/json"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type Broadcaster interface {
	Broadcast(msg []byte)
}

type wsPayload struct {
	Event       string `json:"event"`
	AggregateID uint64 `json:"aggregate_id"`
}

func NewWSBroadcast(b Broadcaster) ports.EventHandler {
	return New("*", func(ctx context.Context, e shared.Event) error {
		msg, err := json.Marshal(wsPayload{
			Event:       e.EventName(),
			AggregateID: e.AggregateID(),
		})
		if err != nil {
			return err
		}
		b.Broadcast(msg)
		return nil
	})
}
