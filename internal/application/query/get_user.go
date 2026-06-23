package query

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
)

type GetUser struct {
	reads ports.RepositoryProvider
}

func NewGetUser(reads ports.RepositoryProvider) *GetUser {
	return &GetUser{reads: reads}
}

func (h *GetUser) Handle(ctx context.Context, id uint64) (*user.User, error) {
	return h.reads.Users().FindByID(ctx, id)
}
