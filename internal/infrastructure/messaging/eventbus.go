package messaging

import (
	"context"
	"sync"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/rs/zerolog"
)

type job struct {
	handler ports.EventHandler
	event   shared.Event
}

type InProcessBus struct {
	workers   int
	queueSize int
	log       zerolog.Logger

	mu       sync.RWMutex
	handlers map[string][]ports.EventHandler

	jobs chan job
	wg   sync.WaitGroup

	startOnce sync.Once
	stopOnce  sync.Once
}

func NewInProcessBus(workers, queueSize int, log zerolog.Logger) *InProcessBus {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 0 {
		queueSize = 0
	}
	return &InProcessBus{
		workers:   workers,
		queueSize: queueSize,
		log:       log,
		handlers:  make(map[string][]ports.EventHandler),
		jobs:      make(chan job, queueSize),
	}
}

func (b *InProcessBus) Subscribe(handler ports.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	name := handler.EventName()
	b.handlers[name] = append(b.handlers[name], handler)
}

func (b *InProcessBus) Start() {
	b.startOnce.Do(func() {
		for i := 0; i < b.workers; i++ {
			b.wg.Add(1)
			go b.work()
		}
	})
}

func (b *InProcessBus) work() {
	defer b.wg.Done()
	for j := range b.jobs {
		b.dispatch(j.handler, j.event)
	}
}

func (b *InProcessBus) dispatch(handler ports.EventHandler, event shared.Event) {
	defer func() {
		if r := recover(); r != nil {
			b.log.Error().
				Interface("panic", r).
				Str("event", event.EventName()).
				Uint64("aggregate_id", event.AggregateID()).
				Msg("event handler panicked")
		}
	}()
	if err := handler.Handle(context.Background(), event); err != nil {
		b.log.Error().
			Err(err).
			Str("event", event.EventName()).
			Uint64("aggregate_id", event.AggregateID()).
			Msg("event handler failed")
	}
}

func (b *InProcessBus) Publish(ctx context.Context, events ...shared.Event) error {
	for _, event := range events {
		b.mu.RLock()
		handlers := b.handlers[event.EventName()]
		dispatched := make([]ports.EventHandler, len(handlers))
		copy(dispatched, handlers)
		b.mu.RUnlock()

		for _, handler := range dispatched {
			j := job{handler: handler, event: event}
			select {
			case b.jobs <- j:
			default:
				b.dispatch(handler, event)
			}
		}
	}
	return nil
}

func (b *InProcessBus) Stop() {
	b.stopOnce.Do(func() {
		close(b.jobs)
		b.wg.Wait()
	})
}

var _ ports.EventBus = (*InProcessBus)(nil)
