package audit

import (
	"testing"
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func actor(id uint64) *uint64 { return &id }

func TestNewEntry(t *testing.T) {
	tests := []struct {
		name     string
		actorID  *uint64
		action   string
		entity   string
		entityID uint64
		before   string
		after    string
		metadata string
		wantErr  bool
		errCode  string
	}{
		{
			name:     "valid entry with actor",
			actorID:  actor(7),
			action:   "order.placed",
			entity:   "order",
			entityID: 42,
			before:   `{"status":"draft"}`,
			after:    `{"status":"placed"}`,
			metadata: `{"ip":"127.0.0.1"}`,
		},
		{
			name:     "valid entry with nil actor",
			actorID:  nil,
			action:   "system.cleanup",
			entity:   "session",
			entityID: 0,
		},
		{
			name:     "empty action rejected",
			actorID:  actor(1),
			action:   "",
			entity:   "order",
			entityID: 1,
			wantErr:  true,
			errCode:  "audit.action_required",
		},
		{
			name:     "empty entity rejected",
			actorID:  actor(1),
			action:   "order.placed",
			entity:   "",
			entityID: 1,
			wantErr:  true,
			errCode:  "audit.entity_required",
		},
		{
			name:     "empty action takes precedence over empty entity",
			actorID:  nil,
			action:   "",
			entity:   "",
			entityID: 1,
			wantErr:  true,
			errCode:  "audit.action_required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := time.Now().UTC()
			entry, err := NewEntry(tc.actorID, tc.action, tc.entity, tc.entityID, tc.before, tc.after, tc.metadata)
			after := time.Now().UTC()

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, entry)
				domainErr, ok := errs.As(err)
				require.True(t, ok)
				assert.Equal(t, errs.KindValidation, domainErr.Kind)
				assert.Equal(t, tc.errCode, domainErr.Code)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, entry)

			assert.Equal(t, uint64(0), entry.ID())
			assert.Equal(t, tc.actorID, entry.ActorID())
			assert.Equal(t, tc.action, entry.Action())
			assert.Equal(t, tc.entity, entry.Entity())
			assert.Equal(t, tc.entityID, entry.EntityID())
			assert.Equal(t, tc.before, entry.Before())
			assert.Equal(t, tc.after, entry.After())
			assert.Equal(t, tc.metadata, entry.Metadata())
			assert.False(t, entry.CreatedAt().Before(before))
			assert.False(t, entry.CreatedAt().After(after))
			assert.Equal(t, time.UTC, entry.CreatedAt().Location())
		})
	}
}

func TestNewEntryActorIsNotAliased(t *testing.T) {
	id := uint64(99)
	entry, err := NewEntry(&id, "user.login", "user", 99, "", "", "")
	require.NoError(t, err)
	require.NotNil(t, entry.ActorID())
	assert.Equal(t, uint64(99), *entry.ActorID())
}

func TestReconstituteEntry(t *testing.T) {
	createdAt := time.Date(2026, time.June, 23, 10, 30, 0, 0, time.UTC)
	a := actor(5)

	entry := ReconstituteEntry(
		77,
		a,
		"payment.captured",
		"payment",
		1234,
		`{"state":"authorized"}`,
		`{"state":"captured"}`,
		`{"gateway":"stripe"}`,
		createdAt,
	)

	require.NotNil(t, entry)
	assert.Equal(t, uint64(77), entry.ID())
	assert.Equal(t, a, entry.ActorID())
	assert.Equal(t, "payment.captured", entry.Action())
	assert.Equal(t, "payment", entry.Entity())
	assert.Equal(t, uint64(1234), entry.EntityID())
	assert.Equal(t, `{"state":"authorized"}`, entry.Before())
	assert.Equal(t, `{"state":"captured"}`, entry.After())
	assert.Equal(t, `{"gateway":"stripe"}`, entry.Metadata())
	assert.Equal(t, createdAt, entry.CreatedAt())
}

func TestReconstituteEntrySkipsValidation(t *testing.T) {
	entry := ReconstituteEntry(1, nil, "", "", 0, "", "", "", time.Time{})
	require.NotNil(t, entry)
	assert.Equal(t, "", entry.Action())
	assert.Equal(t, "", entry.Entity())
	assert.Nil(t, entry.ActorID())
	assert.True(t, entry.CreatedAt().IsZero())
}
