package user

import (
	"testing"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleValid(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{"customer", RoleCustomer, true},
		{"admin", RoleAdmin, true},
		{"empty", Role(""), false},
		{"unknown", Role("superuser"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.role.Valid())
		})
	}
}

func TestNewUserSuccess(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		passwordHash string
		fullName     string
		role         Role
		wantRole     Role
	}{
		{"explicit customer", "alice@example.com", "hash", "Alice", RoleCustomer, RoleCustomer},
		{"explicit admin", "boss@example.com", "hash", "Boss", RoleAdmin, RoleAdmin},
		{"empty role defaults to customer", "bob@example.com", "hash", "Bob", Role(""), RoleCustomer},
		{"empty full name allowed", "c@d.com", "hash", "", RoleCustomer, RoleCustomer},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := NewUser(tc.email, tc.passwordHash, tc.fullName, tc.role)
			require.NoError(t, err)
			require.NotNil(t, u)
			assert.Equal(t, tc.email, u.Email())
			assert.Equal(t, tc.passwordHash, u.PasswordHash())
			assert.Equal(t, tc.fullName, u.FullName())
			assert.Equal(t, tc.wantRole, u.Role())
			assert.Equal(t, uint64(0), u.ID())
			assert.True(t, u.CreatedAt().IsZero())
		})
	}
}

func TestNewUserRecordsUserRegistered(t *testing.T) {
	u, err := NewUser("alice@example.com", "hash", "Alice", RoleCustomer)
	require.NoError(t, err)
	require.True(t, u.HasPendingEvents())

	events := u.PullEvents()
	require.Len(t, events, 1)

	evt, ok := events[0].(UserRegistered)
	require.True(t, ok)
	assert.Equal(t, "user.registered", evt.EventName())
	assert.Equal(t, "alice@example.com", evt.Email)
	assert.Equal(t, u.ID(), evt.AggregateID())

	assert.False(t, u.HasPendingEvents())
	assert.Empty(t, u.PullEvents())
}

func TestNewUserEventSatisfiesEventInterface(t *testing.T) {
	var _ shared.Event = UserRegistered{}
}

func TestNewUserInvariantRejections(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		passwordHash string
		fullName     string
		role         Role
		wantCode     string
	}{
		{"empty email", "", "hash", "Alice", RoleCustomer, "user.email_required"},
		{"email without at", "aliceexample.com", "hash", "Alice", RoleCustomer, "user.email_invalid"},
		{"empty password hash", "alice@example.com", "", "Alice", RoleCustomer, "user.password_required"},
		{"invalid role", "alice@example.com", "hash", "Alice", Role("ghost"), "user.role_invalid"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := NewUser(tc.email, tc.passwordHash, tc.fullName, tc.role)
			require.Error(t, err)
			require.Nil(t, u)

			domainErr, ok := errs.As(err)
			require.True(t, ok)
			assert.Equal(t, errs.KindValidation, domainErr.Kind)
			assert.Equal(t, tc.wantCode, domainErr.Code)
		})
	}
}

func TestReconstituteUser(t *testing.T) {
	createdAt := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	u := ReconstituteUser(42, "admin@example.com", "stored-hash", "Admin", RoleAdmin, createdAt)

	require.NotNil(t, u)
	assert.Equal(t, uint64(42), u.ID())
	assert.Equal(t, "admin@example.com", u.Email())
	assert.Equal(t, "stored-hash", u.PasswordHash())
	assert.Equal(t, "Admin", u.FullName())
	assert.Equal(t, RoleAdmin, u.Role())
	assert.Equal(t, createdAt, u.CreatedAt())

	assert.False(t, u.HasPendingEvents())
	assert.Empty(t, u.PullEvents())
}

func TestUpdateProfile(t *testing.T) {
	u, err := NewUser("alice@example.com", "hash", "Alice", RoleCustomer)
	require.NoError(t, err)
	u.PullEvents()

	u.UpdateProfile("Alice Cooper")
	assert.Equal(t, "Alice Cooper", u.FullName())

	u.UpdateProfile("")
	assert.Equal(t, "", u.FullName())

	assert.False(t, u.HasPendingEvents())
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want bool
	}{
		{"admin is admin", RoleAdmin, true},
		{"customer is not admin", RoleCustomer, false},
		{"default is not admin", Role(""), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := NewUser("alice@example.com", "hash", "Alice", tc.role)
			require.NoError(t, err)
			assert.Equal(t, tc.want, u.IsAdmin())
		})
	}
}
