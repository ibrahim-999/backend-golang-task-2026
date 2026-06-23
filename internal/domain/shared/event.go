package shared

type Event interface {
	EventName() string
	AggregateID() uint64
}

type AggregateRoot struct {
	pending []Event
}

func (a *AggregateRoot) Record(events ...Event) {
	a.pending = append(a.pending, events...)
}

func (a *AggregateRoot) PullEvents() []Event {
	out := a.pending
	a.pending = nil
	return out
}

func (a *AggregateRoot) PendingEvents() []Event {
	return a.pending
}

func (a *AggregateRoot) HasPendingEvents() bool {
	return len(a.pending) > 0
}
