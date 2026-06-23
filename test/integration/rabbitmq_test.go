//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/messaging"
)

type capturingHandler struct {
	name  string
	fired chan struct{}
}

func (h capturingHandler) EventName() string { return h.name }

func (h capturingHandler) Handle(ctx context.Context, e shared.Event) error {
	select {
	case h.fired <- struct{}{}:
	default:
	}
	return nil
}

func TestRabbitMQPublishConsumeAndCompositeBus(t *testing.T) {
	host := getenv("RABBITMQ_HOST", "rabbitmq")
	user := getenv("RABBITMQ_USER", "orders")
	pass := getenv("RABBITMQ_PASSWORD", "ordersecret")
	url := fmt.Sprintf("amqp://%s:%s@%s:5672/", user, pass, host)
	exchange := "orders.events.test"

	publisher, err := messaging.NewRabbitPublisher(url, exchange, zerolog.Nop())
	if err != nil {
		t.Skipf("rabbitmq broker not reachable, skipping: %v", err)
	}
	defer func() { _ = publisher.Close() }()

	consumer, err := messaging.NewRabbitConsumer(url, exchange, zerolog.Nop())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, consumer.Start(ctx))

	event := order.OrderPlaced{OrderID: 1, UserID: 1, TotalAmount: 1000, Currency: "USD"}
	require.NoError(t, publisher.Publish(context.Background(), event))

	time.Sleep(300 * time.Millisecond)
	cancel()
	consumer.Stop()

	inproc := messaging.NewInProcessBus(2, 16, zerolog.Nop())
	composite := messaging.NewCompositeBus(inproc, publisher)
	composite.Start()
	defer composite.Stop()

	fired := make(chan struct{}, 1)
	composite.Subscribe(capturingHandler{name: event.EventName(), fired: fired})
	require.NoError(t, composite.Publish(context.Background(), event))

	select {
	case <-fired:
	case <-time.After(2 * time.Second):
		t.Fatal("composite bus did not dispatch the event to the in-process handler")
	}
}
