package notification

type NotificationQueued struct {
	NotificationID uint64
}

func (e NotificationQueued) EventName() string {
	return "notification.queued"
}

func (e NotificationQueued) AggregateID() uint64 {
	return e.NotificationID
}

type NotificationSent struct {
	NotificationID uint64
}

func (e NotificationSent) EventName() string {
	return "notification.sent"
}

func (e NotificationSent) AggregateID() uint64 {
	return e.NotificationID
}

type NotificationFailed struct {
	NotificationID uint64
	Reason         string
}

func (e NotificationFailed) EventName() string {
	return "notification.failed"
}

func (e NotificationFailed) AggregateID() uint64 {
	return e.NotificationID
}
