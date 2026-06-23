package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

func TestNewMoney(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		m, err := shared.NewMoney(1599, "USD")
		require.NoError(t, err)
		assert.Equal(t, int64(1599), m.Amount())
		assert.Equal(t, "USD", m.Currency())
	})

	t.Run("negative amount rejected", func(t *testing.T) {
		_, err := shared.NewMoney(-1, "USD")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
	})

	t.Run("invalid currency rejected", func(t *testing.T) {
		_, err := shared.NewMoney(100, "DOLLAR")
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
	})
}

func TestMustMoney(t *testing.T) {
	assert.NotPanics(t, func() { shared.MustMoney(100, "USD") })
	assert.Panics(t, func() { shared.MustMoney(-1, "USD") })
}

func TestMoneyArithmetic(t *testing.T) {
	t.Run("add same currency", func(t *testing.T) {
		a := shared.MustMoney(1000, "USD")
		b := shared.MustMoney(250, "USD")
		sum, err := a.Add(b)
		require.NoError(t, err)
		assert.Equal(t, int64(1250), sum.Amount())
	})

	t.Run("add different currency fails", func(t *testing.T) {
		a := shared.MustMoney(1000, "USD")
		b := shared.MustMoney(250, "EUR")
		_, err := a.Add(b)
		require.Error(t, err)
		assert.Equal(t, errs.KindValidation, errs.KindOf(err))
	})

	t.Run("multiply by quantity", func(t *testing.T) {
		price := shared.MustMoney(599, "USD")
		assert.Equal(t, int64(1797), price.Mul(3).Amount())
	})
}

func TestMoneyComparisons(t *testing.T) {
	a := shared.MustMoney(1000, "USD")
	b := shared.MustMoney(1000, "USD")
	c := shared.MustMoney(500, "USD")
	eur := shared.MustMoney(1000, "EUR")

	assert.True(t, a.Equal(b))
	assert.False(t, a.Equal(c))
	assert.False(t, a.Equal(eur))
	assert.True(t, a.GreaterThan(c))
	assert.False(t, c.GreaterThan(a))
	assert.False(t, a.GreaterThan(eur))
	assert.True(t, shared.ZeroMoney("USD").IsZero())
	assert.False(t, a.IsZero())
}

func TestMoneyString(t *testing.T) {
	assert.Equal(t, "15.99 USD", shared.MustMoney(1599, "USD").String())
}
