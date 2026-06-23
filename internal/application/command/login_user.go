package command

import (
	"context"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type LoginUser struct {
	uow    ports.UnitOfWork
	reads  ports.RepositoryProvider
	hasher ports.PasswordHasher
	tokens ports.TokenIssuer
}

func NewLoginUser(uow ports.UnitOfWork, reads ports.RepositoryProvider, hasher ports.PasswordHasher, tokens ports.TokenIssuer) *LoginUser {
	return &LoginUser{uow: uow, reads: reads, hasher: hasher, tokens: tokens}
}

func (h *LoginUser) Handle(ctx context.Context, email, password string) (string, *user.User, error) {
	u, err := h.reads.Users().FindByEmail(ctx, email)
	if err != nil {
		if errs.KindOf(err) == errs.KindNotFound {
			return "", nil, errs.Unauthorized("auth.invalid_credentials", "invalid email or password")
		}
		return "", nil, err
	}

	if err := h.hasher.Compare(u.PasswordHash(), password); err != nil {
		return "", nil, errs.Unauthorized("auth.invalid_credentials", "invalid email or password")
	}

	token, err := h.tokens.Issue(u.ID(), string(u.Role()))
	if err != nil {
		return "", nil, err
	}

	return token, u, nil
}
