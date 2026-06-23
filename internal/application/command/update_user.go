package command

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
)

type UpdateUser struct {
	uow ports.UnitOfWork
}

func NewUpdateUser(uow ports.UnitOfWork) *UpdateUser {
	return &UpdateUser{uow: uow}
}

func (h *UpdateUser) Handle(ctx context.Context, id uint64, fullName string) (*user.User, error) {
	var updated *user.User
	err := h.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		u, err := rp.Users().FindByID(ctx, id)
		if err != nil {
			return err
		}
		u.UpdateProfile(fullName)
		if err := rp.Users().Update(ctx, u); err != nil {
			return err
		}
		updated = u
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
