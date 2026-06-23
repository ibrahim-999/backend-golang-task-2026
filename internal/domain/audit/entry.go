package audit

import (
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Entry struct {
	id        uint64
	actorID   *uint64
	action    string
	entity    string
	entityID  uint64
	before    string
	after     string
	metadata  string
	createdAt time.Time
}

func NewEntry(actorID *uint64, action, entity string, entityID uint64, before, after, metadata string) (*Entry, error) {
	if action == "" {
		return nil, errs.Validation("audit.action_required", "action must not be empty")
	}
	if entity == "" {
		return nil, errs.Validation("audit.entity_required", "entity must not be empty")
	}
	return &Entry{
		actorID:   actorID,
		action:    action,
		entity:    entity,
		entityID:  entityID,
		before:    before,
		after:     after,
		metadata:  metadata,
		createdAt: time.Now().UTC(),
	}, nil
}

func ReconstituteEntry(id uint64, actorID *uint64, action, entity string, entityID uint64, before, after, metadata string, createdAt time.Time) *Entry {
	return &Entry{
		id:        id,
		actorID:   actorID,
		action:    action,
		entity:    entity,
		entityID:  entityID,
		before:    before,
		after:     after,
		metadata:  metadata,
		createdAt: createdAt,
	}
}

func (e *Entry) ID() uint64 { return e.id }

func (e *Entry) ActorID() *uint64 { return e.actorID }

func (e *Entry) Action() string { return e.action }

func (e *Entry) Entity() string { return e.entity }

func (e *Entry) EntityID() uint64 { return e.entityID }

func (e *Entry) Before() string { return e.before }

func (e *Entry) After() string { return e.after }

func (e *Entry) Metadata() string { return e.metadata }

func (e *Entry) CreatedAt() time.Time { return e.createdAt }
