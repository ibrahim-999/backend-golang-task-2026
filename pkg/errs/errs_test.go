package errs_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

func TestConstructorsAndKinds(t *testing.T) {
	cases := []struct {
		err  *errs.Error
		kind errs.Kind
	}{
		{errs.Validation("v", "m"), errs.KindValidation},
		{errs.NotFound("v", "m"), errs.KindNotFound},
		{errs.Conflict("v", "m"), errs.KindConflict},
		{errs.Unauthorized("v", "m"), errs.KindUnauthorized},
		{errs.Forbidden("v", "m"), errs.KindForbidden},
		{errs.OutOfStock("v", "m"), errs.KindOutOfStock},
		{errs.PaymentFailed("v", "m"), errs.KindPayment},
		{errs.Internal("v", "m"), errs.KindInternal},
		{errs.Unavailable("v", "m"), errs.KindUnavailable},
	}
	for _, c := range cases {
		assert.Equal(t, c.kind, errs.KindOf(c.err))
		assert.NotEmpty(t, c.err.Error())
		got, ok := errs.As(c.err)
		assert.True(t, ok)
		assert.Equal(t, c.err.Code, got.Code)
	}
}

func TestWithCauseAndUnwrap(t *testing.T) {
	root := errors.New("root cause")
	wrapped := errs.Internal("internal", "boom").WithCause(root)
	assert.ErrorIs(t, wrapped, root)
	assert.Contains(t, wrapped.Error(), "root cause")

	assert.Equal(t, errs.KindInternal, errs.KindOf(errors.New("plain")))
	_, ok := errs.As(errors.New("plain"))
	assert.False(t, ok)
}
