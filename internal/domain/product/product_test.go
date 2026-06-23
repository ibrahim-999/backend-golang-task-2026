package product

import (
	"testing"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProduct(t *testing.T) {
	price := shared.MustMoney(1999, "USD")

	tests := []struct {
		name        string
		sku         string
		productName string
		wantErr     bool
		wantCode    string
	}{
		{
			name:        "valid product",
			sku:         "SKU-1",
			productName: "Widget",
			wantErr:     false,
		},
		{
			name:        "empty sku rejected",
			sku:         "",
			productName: "Widget",
			wantErr:     true,
			wantCode:    "product.sku_required",
		},
		{
			name:        "empty name rejected",
			sku:         "SKU-1",
			productName: "",
			wantErr:     true,
			wantCode:    "product.name_required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewProduct(tc.sku, tc.productName, "desc", price)

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, p)
				domainErr, ok := errs.As(err)
				require.True(t, ok)
				assert.Equal(t, errs.KindValidation, domainErr.Kind)
				assert.Equal(t, tc.wantCode, domainErr.Code)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, tc.sku, p.SKU())
			assert.Equal(t, tc.productName, p.Name())
			assert.Equal(t, "desc", p.Description())
			assert.True(t, p.Price().Equal(price))
			assert.True(t, p.Active())
			assert.False(t, p.CreatedAt().IsZero())
		})
	}
}

func TestNewProductRecordsCreatedEvent(t *testing.T) {
	p, err := NewProduct("SKU-9", "Widget", "desc", shared.MustMoney(500, "USD"))
	require.NoError(t, err)
	require.True(t, p.HasPendingEvents())

	events := p.PullEvents()
	require.Len(t, events, 1)

	created, ok := events[0].(ProductCreated)
	require.True(t, ok)
	assert.Equal(t, "product.created", created.EventName())
	assert.Equal(t, "SKU-9", created.SKU)
	assert.Equal(t, p.ID(), created.AggregateID())

	assert.False(t, p.HasPendingEvents())
}

func TestReconstituteProduct(t *testing.T) {
	createdAt := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	price := shared.MustMoney(2500, "USD")

	p := ReconstituteProduct(42, "SKU-42", "Gadget", "info", price, false, createdAt)

	require.NotNil(t, p)
	assert.Equal(t, uint64(42), p.ID())
	assert.Equal(t, "SKU-42", p.SKU())
	assert.Equal(t, "Gadget", p.Name())
	assert.Equal(t, "info", p.Description())
	assert.True(t, p.Price().Equal(price))
	assert.False(t, p.Active())
	assert.Equal(t, createdAt, p.CreatedAt())
	assert.False(t, p.HasPendingEvents())
}

func TestProductRename(t *testing.T) {
	tests := []struct {
		name     string
		newName  string
		wantErr  bool
		wantName string
	}{
		{
			name:     "valid rename",
			newName:  "New Name",
			wantErr:  false,
			wantName: "New Name",
		},
		{
			name:     "empty name rejected",
			newName:  "",
			wantErr:  true,
			wantName: "Original",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := ReconstituteProduct(1, "SKU-1", "Original", "desc", shared.MustMoney(100, "USD"), true, time.Now())

			err := p.Rename(tc.newName)

			if tc.wantErr {
				require.Error(t, err)
				domainErr, ok := errs.As(err)
				require.True(t, ok)
				assert.Equal(t, errs.KindValidation, domainErr.Kind)
				assert.Equal(t, "product.name_required", domainErr.Code)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.wantName, p.Name())
		})
	}
}

func TestProductReprice(t *testing.T) {
	p := ReconstituteProduct(7, "SKU-7", "Thing", "desc", shared.MustMoney(100, "USD"), true, time.Now())
	newPrice := shared.MustMoney(750, "USD")

	p.Reprice(newPrice)

	assert.True(t, p.Price().Equal(newPrice))

	events := p.PullEvents()
	require.Len(t, events, 1)

	repriced, ok := events[0].(ProductRepriced)
	require.True(t, ok)
	assert.Equal(t, "product.repriced", repriced.EventName())
	assert.Equal(t, uint64(7), repriced.AggregateID())
}

func TestProductActivateDeactivate(t *testing.T) {
	p := ReconstituteProduct(3, "SKU-3", "Thing", "desc", shared.MustMoney(100, "USD"), true, time.Now())

	p.Deactivate()
	assert.False(t, p.Active())

	p.Activate()
	assert.True(t, p.Active())

	assert.False(t, p.HasPendingEvents())
}
