package auth

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

func (h *BcryptHasher) Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", errs.Internal("password_hash_failed", "failed to hash password")
	}
	return string(bytes), nil
}

func (h *BcryptHasher) Compare(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return errs.Unauthorized("invalid_credentials", "invalid credentials")
	}
	return nil
}

var _ ports.PasswordHasher = (*BcryptHasher)(nil)
