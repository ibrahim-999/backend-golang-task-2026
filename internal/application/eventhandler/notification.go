package eventhandler

import (
	"context"
	"fmt"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	domnotification "github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/order"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	notificationinfra "github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/notification"
)

func NotificationOnOrderPaid(orders ports.OrderRepository, dispatcher *notificationinfra.Dispatcher) ports.EventHandler {
	return New(order.OrderPaid{}.EventName(), func(ctx context.Context, e shared.Event) error {
		o, err := orders.FindByID(ctx, e.AggregateID())
		if err != nil {
			return err
		}
		userID := o.UserID()
		subject := fmt.Sprintf("Order #%d confirmed", o.ID())
		n, err := domnotification.NewNotification(&userID, domnotification.TypeOrderConfirmed, domnotification.ChannelEmail, subject, "")
		if err != nil {
			return err
		}
		return dispatcher.Enqueue(ctx, n)
	})
}

func NotificationOnOrderFailed(orders ports.OrderRepository, dispatcher *notificationinfra.Dispatcher) ports.EventHandler {
	return New(order.OrderFailed{}.EventName(), func(ctx context.Context, e shared.Event) error {
		o, err := orders.FindByID(ctx, e.AggregateID())
		if err != nil {
			return err
		}
		userID := o.UserID()
		subject := fmt.Sprintf("Order #%d failed", o.ID())
		n, err := domnotification.NewNotification(&userID, domnotification.TypeOrderFailed, domnotification.ChannelEmail, subject, o.FailureReason())
		if err != nil {
			return err
		}
		return dispatcher.Enqueue(ctx, n)
	})
}
