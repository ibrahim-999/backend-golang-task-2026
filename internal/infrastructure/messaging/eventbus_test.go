package messaging_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/messaging"
)

type recordingHandler struct {
	name string
	mu   *sync.Mutex
	seen *[]string
}

func (h recordingHandler) EventName() string { return h.name }

func (h recordingHandler) Handle(ctx context.Context, e shared.Event) error {
	h.mu.Lock()
	*h.seen = append(*h.seen, e.EventName())
	h.mu.Unlock()
	return nil
}

func TestInProcessBusDispatchesToNamedAndWildcard(t *testing.T) {
	bus := messaging.NewInProcessBus(3, 32, zerolog.Nop())
	bus.Start()
	defer bus.Stop()

	var mu sync.Mutex
	var seen []string

	bus.Subscribe(recordingHandler{name: "order.paid", mu: &mu, seen: &seen})
	bus.Subscribe(recordingHandler{name: "*", mu: &mu, seen: &seen})

	require.NoError(t, bus.Publish(context.Background(),
		order.OrderPaid{OrderID: 1},
		order.OrderFulfilled{OrderID: 1},
	))

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(seen) >= 3
	}, 2*time.Second, 10*time.Millisecond)
}
