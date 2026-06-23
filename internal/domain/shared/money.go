package shared

import (
	"fmt"

	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

const DefaultCurrency = "USD"

type Money struct {
	amount   int64
	currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, errs.Validation("money.negative", "amount cannot be negative")
	}
	if len(currency) != 3 {
		return Money{}, errs.Validation("money.currency", fmt.Sprintf("invalid currency %q", currency))
	}
	return Money{amount: amount, currency: currency}, nil
}

func MustMoney(amount int64, currency string) Money {
	m, err := NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

func ZeroMoney(currency string) Money {
	return Money{amount: 0, currency: currency}
}

func (m Money) Amount() int64    { return m.amount }
func (m Money) Currency() string { return m.currency }
func (m Money) IsZero() bool     { return m.amount == 0 }

func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, errs.Validation("money.currency_mismatch", "cannot add different currencies")
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

func (m Money) Mul(quantity int64) Money {
	return Money{amount: m.amount * quantity, currency: m.currency}
}

func (m Money) Equal(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}

func (m Money) GreaterThan(other Money) bool {
	return m.currency == other.currency && m.amount > other.amount
}

func (m Money) String() string {
	return fmt.Sprintf("%d.%02d %s", m.amount/100, m.amount%100, m.currency)
}
