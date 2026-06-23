package payment

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type MockGateway struct {
	failureRate float64
	minLatency  time.Duration
	maxLatency  time.Duration

	mu      sync.Mutex
	results map[string]ports.ChargeResult

	rngMu sync.Mutex
	rng   *rand.Rand
}

func NewMockGateway(failureRate float64, minLatency, maxLatency time.Duration) *MockGateway {
	return &MockGateway{
		failureRate: failureRate,
		minLatency:  minLatency,
		maxLatency:  maxLatency,
		results:     make(map[string]ports.ChargeResult),
		rng:         rand.New(rand.NewSource(1)),
	}
}

func (g *MockGateway) randFloat() float64 {
	g.rngMu.Lock()
	defer g.rngMu.Unlock()
	return g.rng.Float64()
}

func (g *MockGateway) randLatency() time.Duration {
	if g.maxLatency <= g.minLatency {
		return g.minLatency
	}
	g.rngMu.Lock()
	defer g.rngMu.Unlock()
	span := int64(g.maxLatency - g.minLatency)
	return g.minLatency + time.Duration(g.rng.Int63n(span))
}

func (g *MockGateway) Charge(ctx context.Context, req ports.ChargeRequest) (ports.ChargeResult, error) {
	g.mu.Lock()
	if existing, ok := g.results[req.IdempotencyKey]; ok {
		g.mu.Unlock()
		return existing, nil
	}
	g.mu.Unlock()

	timer := time.NewTimer(g.randLatency())
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ports.ChargeResult{}, ctx.Err()
	case <-timer.C:
	}

	if g.randFloat() < g.failureRate {
		return ports.ChargeResult{}, errs.PaymentFailed("payment.declined", "payment was declined by the gateway")
	}

	result := ports.ChargeResult{
		Provider:    "mock",
		ProviderRef: uuid.NewString(),
	}

	g.mu.Lock()
	if existing, ok := g.results[req.IdempotencyKey]; ok {
		g.mu.Unlock()
		return existing, nil
	}
	g.results[req.IdempotencyKey] = result
	g.mu.Unlock()

	return result, nil
}

var _ ports.PaymentGateway = (*MockGateway)(nil)
