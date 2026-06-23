package command

import (
	"context"
	"strings"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type RegisterUser struct {
	uow    ports.UnitOfWork
	hasher ports.PasswordHasher
}

func NewRegisterUser(uow ports.UnitOfWork, hasher ports.PasswordHasher) *RegisterUser {
	return &RegisterUser{uow: uow, hasher: hasher}
}

type RegisterUserInput struct {
	Email    string
	Password string
	FullName string
	Role     string
}

func (h *RegisterUser) Handle(ctx context.Context, in RegisterUserInput) (*user.User, error) {
	hash, err := h.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	u, err := user.NewUser(in.Email, hash, in.FullName, user.Role(in.Role))
	if err != nil {
		return nil, err
	}

	err = h.uow.Do(ctx, func(rp ports.RepositoryProvider) error {
		if cerr := rp.Users().Create(ctx, u); cerr != nil {
			if isDuplicate(cerr) {
				return errs.Conflict("user.email_taken", "email is already registered").WithCause(cerr)
			}
			return cerr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return u, nil
}

func isDuplicate(err error) bool {
	if errs.KindOf(err) == errs.KindConflict {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "1062")
}
