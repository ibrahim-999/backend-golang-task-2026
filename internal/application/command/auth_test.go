package command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/user"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type fakeHasher struct {
	prefix string
}

func (h *fakeHasher) Hash(plain string) (string, error) {
	return h.prefix + plain, nil
}

func (h *fakeHasher) Compare(hash, plain string) error {
	if hash != h.prefix+plain {
		return errs.Unauthorized("auth.mismatch", "password does not match")
	}
	return nil
}

type fakeTokenIssuer struct {
	lastUserID uint64
	lastRole   string
	token      string
}

func (i *fakeTokenIssuer) Issue(userID uint64, role string) (string, error) {
	i.lastUserID = userID
	i.lastRole = role
	if i.token == "" {
		i.token = "issued-token"
	}
	return i.token, nil
}

func TestRegisterUserHashesAndStores(t *testing.T) {
	store := newFakeStore()
	hasher := &fakeHasher{prefix: "hashed:"}

	h := NewRegisterUser(&fakeUoW{store: store}, hasher)

	u, err := h.Handle(context.Background(), RegisterUserInput{
		Email:    "alice@example.com",
		Password: "s3cret",
		FullName: "Alice",
		Role:     string(user.RoleCustomer),
	})

	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "alice@example.com", u.Email())
	assert.Equal(t, "hashed:s3cret", u.PasswordHash())
	assert.Equal(t, user.RoleCustomer, u.Role())

	store.mu.Lock()
	stored := store.users[u.ID()]
	store.mu.Unlock()
	require.NotNil(t, stored)
	assert.Equal(t, "hashed:s3cret", stored.PasswordHash())
}

func TestRegisterUserDuplicateEmailConflict(t *testing.T) {
	store := newFakeStore()
	hasher := &fakeHasher{prefix: "hashed:"}

	h := NewRegisterUser(&fakeUoW{store: store}, hasher)

	in := RegisterUserInput{
		Email:    "dup@example.com",
		Password: "pw",
		FullName: "Dup",
		Role:     string(user.RoleCustomer),
	}

	_, err := h.Handle(context.Background(), in)
	require.NoError(t, err)

	_, err = h.Handle(context.Background(), in)
	require.Error(t, err)
	assert.Equal(t, errs.KindConflict, errs.KindOf(err))
}

func TestLoginUserReturnsToken(t *testing.T) {
	store := newFakeStore()
	hasher := &fakeHasher{prefix: "hashed:"}
	issuer := &fakeTokenIssuer{token: "jwt-123"}
	provider := &fakeProvider{store: store}

	register := NewRegisterUser(&fakeUoW{store: store}, hasher)
	u, err := register.Handle(context.Background(), RegisterUserInput{
		Email:    "bob@example.com",
		Password: "correct-horse",
		FullName: "Bob",
		Role:     string(user.RoleAdmin),
	})
	require.NoError(t, err)

	login := NewLoginUser(&fakeUoW{store: store}, provider, hasher, issuer)

	token, got, err := login.Handle(context.Background(), "bob@example.com", "correct-horse")
	require.NoError(t, err)
	assert.Equal(t, "jwt-123", token)
	require.NotNil(t, got)
	assert.Equal(t, u.ID(), got.ID())
	assert.Equal(t, u.ID(), issuer.lastUserID)
	assert.Equal(t, string(user.RoleAdmin), issuer.lastRole)
}

func TestLoginUserWrongPasswordUnauthorized(t *testing.T) {
	store := newFakeStore()
	hasher := &fakeHasher{prefix: "hashed:"}
	issuer := &fakeTokenIssuer{}
	provider := &fakeProvider{store: store}

	register := NewRegisterUser(&fakeUoW{store: store}, hasher)
	_, err := register.Handle(context.Background(), RegisterUserInput{
		Email:    "carol@example.com",
		Password: "right-password",
		FullName: "Carol",
		Role:     string(user.RoleCustomer),
	})
	require.NoError(t, err)

	login := NewLoginUser(&fakeUoW{store: store}, provider, hasher, issuer)

	token, got, err := login.Handle(context.Background(), "carol@example.com", "wrong-password")
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, got)
	assert.Equal(t, errs.KindUnauthorized, errs.KindOf(err))
}

func TestLoginUserUnknownEmailUnauthorized(t *testing.T) {
	store := newFakeStore()
	hasher := &fakeHasher{prefix: "hashed:"}
	issuer := &fakeTokenIssuer{}
	provider := &fakeProvider{store: store}

	login := NewLoginUser(&fakeUoW{store: store}, provider, hasher, issuer)

	token, got, err := login.Handle(context.Background(), "nobody@example.com", "whatever")
	require.Error(t, err)
	assert.Empty(t, token)
	assert.Nil(t, got)
	assert.Equal(t, errs.KindUnauthorized, errs.KindOf(err))
}
