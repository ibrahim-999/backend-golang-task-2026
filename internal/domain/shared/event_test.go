package shared_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
)

type fakeEvent struct {
	name string
	id   uint64
}

func (e fakeEvent) EventName() string   { return e.name }
func (e fakeEvent) AggregateID() uint64 { return e.id }

func TestAggregateRootRecordsAndPulls(t *testing.T) {
	var root shared.AggregateRoot
	assert.False(t, root.HasPendingEvents())

	root.Record(fakeEvent{name: "a", id: 1})
	root.Record(fakeEvent{name: "b", id: 1}, fakeEvent{name: "c", id: 1})

	assert.True(t, root.HasPendingEvents())
	assert.Len(t, root.PendingEvents(), 3)

	pulled := root.PullEvents()
	assert.Len(t, pulled, 3)
	assert.Equal(t, "a", pulled[0].EventName())

	assert.False(t, root.HasPendingEvents())
	assert.Empty(t, root.PullEvents())
}
