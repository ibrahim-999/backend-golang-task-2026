package ports

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type ChargeRequest struct {
	IdempotencyKey string
	Amount         shared.Money
}

type ChargeResult struct {
	Provider    string
	ProviderRef string
}

type PaymentGateway interface {
	Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
}
