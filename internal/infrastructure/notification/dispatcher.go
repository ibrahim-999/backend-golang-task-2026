package notificationinfra

import (
	"context"
	"sync"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	domnotification "github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	"github.com/rs/zerolog"
)

type Sender interface {
	Send(ctx context.Context, n *domnotification.Notification) error
}

type LogSender struct {
	log zerolog.Logger
}

func NewLogSender(log zerolog.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) Send(ctx context.Context, n *domnotification.Notification) error {
	s.log.Info().
		Uint64("notification_id", n.ID()).
		Str("type", string(n.Type())).
		Str("channel", string(n.Channel())).
		Str("subject", n.Subject()).
		Msg("notification sent")
	return nil
}

type Dispatcher struct {
	repo      ports.NotificationRepository
	sender    Sender
	workers   int
	queueSize int
	log       zerolog.Logger

	queue chan *domnotification.Notification
	wg    sync.WaitGroup

	startOnce sync.Once
	stopOnce  sync.Once
}

func NewDispatcher(repo ports.NotificationRepository, sender Sender, workers, queueSize int, log zerolog.Logger) *Dispatcher {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 0 {
		queueSize = 0
	}
	return &Dispatcher{
		repo:      repo,
		sender:    sender,
		workers:   workers,
		queueSize: queueSize,
		log:       log,
		queue:     make(chan *domnotification.Notification, queueSize),
	}
}

func (d *Dispatcher) Start() {
	d.startOnce.Do(func() {
		for i := 0; i < d.workers; i++ {
			d.wg.Add(1)
			go d.worker(i)
		}
	})
}

func (d *Dispatcher) Enqueue(ctx context.Context, n *domnotification.Notification) error {
	if err := d.repo.Create(ctx, n); err != nil {
		d.log.Error().Err(err).
			Uint64("notification_id", n.ID()).
			Msg("failed to persist notification")
		return err
	}

	select {
	case d.queue <- n:
	default:
		d.log.Warn().
			Uint64("notification_id", n.ID()).
			Msg("dispatch queue full, notification left persisted for later sweep")
	}
	return nil
}

func (d *Dispatcher) worker(id int) {
	defer d.wg.Done()
	for n := range d.queue {
		d.process(id, n)
	}
}

func (d *Dispatcher) process(id int, n *domnotification.Notification) {
	defer func() {
		if r := recover(); r != nil {
			d.log.Error().
				Int("worker", id).
				Uint64("notification_id", n.ID()).
				Interface("panic", r).
				Msg("recovered from panic while dispatching notification")
		}
	}()

	ctx := context.Background()
	n.IncrementAttempt()

	if err := d.sender.Send(ctx, n); err != nil {
		d.log.Error().Err(err).
			Int("worker", id).
			Uint64("notification_id", n.ID()).
			Msg("notification send failed")
		if markErr := n.MarkFailed(err.Error()); markErr != nil {
			d.log.Error().Err(markErr).
				Uint64("notification_id", n.ID()).
				Msg("failed to mark notification failed")
			return
		}
		if updErr := d.repo.Update(ctx, n); updErr != nil {
			d.log.Error().Err(updErr).
				Uint64("notification_id", n.ID()).
				Msg("failed to persist failed notification")
		}
		return
	}

	if markErr := n.MarkSent(); markErr != nil {
		d.log.Error().Err(markErr).
			Uint64("notification_id", n.ID()).
			Msg("failed to mark notification sent")
		return
	}
	if updErr := d.repo.Update(ctx, n); updErr != nil {
		d.log.Error().Err(updErr).
			Uint64("notification_id", n.ID()).
			Msg("failed to persist sent notification")
	}
}

func (d *Dispatcher) Stop() {
	d.stopOnce.Do(func() {
		close(d.queue)
		d.wg.Wait()
	})
}
