package notification

import (
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/domain/shared"
	"github.com/ibrahim-999/backend-golang-task-2026/pkg/errs"
)

type Type string

const (
	TypeOrderConfirmed Type = "order_confirmed"
	TypeOrderFailed    Type = "order_failed"
	TypePaymentFailed  Type = "payment_failed"
	TypeLowStock       Type = "low_stock"
)

type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelSMS   Channel = "sms"
	ChannelPush  Channel = "push"
)

type Status string

const (
	StatusQueued Status = "queued"
	StatusSent   Status = "sent"
	StatusFailed Status = "failed"
)

type Notification struct {
	shared.AggregateRoot
	id            uint64
	userID        *uint64
	ntype         Type
	channel       Channel
	status        Status
	subject       string
	payload       string
	attempts      int
	failureReason string
	createdAt     time.Time
}

func NewNotification(userID *uint64, ntype Type, channel Channel, subject, payload string) (*Notification, error) {
	if !ntype.valid() {
		return nil, errs.Validation("notification.type.invalid", "notification type is invalid")
	}
	if !channel.valid() {
		return nil, errs.Validation("notification.channel.invalid", "notification channel is invalid")
	}
	if subject == "" {
		return nil, errs.Validation("notification.subject.required", "notification subject must not be empty")
	}

	n := &Notification{
		userID:  userID,
		ntype:   ntype,
		channel: channel,
		status:  StatusQueued,
		subject: subject,
		payload: payload,
	}
	n.Record(NotificationQueued{NotificationID: n.id})
	return n, nil
}

func ReconstituteNotification(id uint64, userID *uint64, ntype Type, channel Channel, status Status, subject, payload string, attempts int, failureReason string, createdAt time.Time) *Notification {
	return &Notification{
		id:            id,
		userID:        userID,
		ntype:         ntype,
		channel:       channel,
		status:        status,
		subject:       subject,
		payload:       payload,
		attempts:      attempts,
		failureReason: failureReason,
		createdAt:     createdAt,
	}
}

func (n *Notification) IncrementAttempt() {
	n.attempts++
}

func (n *Notification) MarkSent() error {
	if n.status != StatusQueued && n.status != StatusFailed {
		return errs.Conflict("notification.send.invalid_status", "notification can only be sent from queued or failed status")
	}
	n.status = StatusSent
	n.failureReason = ""
	n.Record(NotificationSent{NotificationID: n.id})
	return nil
}

func (n *Notification) MarkFailed(reason string) error {
	if reason == "" {
		return errs.Validation("notification.fail.reason_required", "failure reason must not be empty")
	}
	if n.status == StatusSent {
		return errs.Conflict("notification.fail.already_sent", "a sent notification cannot be marked failed")
	}
	n.status = StatusFailed
	n.failureReason = reason
	n.Record(NotificationFailed{NotificationID: n.id, Reason: reason})
	return nil
}

func (n *Notification) AssignID(id uint64) {
	if n.id == 0 {
		n.id = id
	}
}

func (n *Notification) ID() uint64 {
	return n.id
}

func (n *Notification) UserID() *uint64 {
	return n.userID
}

func (n *Notification) Type() Type {
	return n.ntype
}

func (n *Notification) Channel() Channel {
	return n.channel
}

func (n *Notification) Status() Status {
	return n.status
}

func (n *Notification) Subject() string {
	return n.subject
}

func (n *Notification) Payload() string {
	return n.payload
}

func (n *Notification) Attempts() int {
	return n.attempts
}

func (n *Notification) FailureReason() string {
	return n.failureReason
}

func (n *Notification) CreatedAt() time.Time {
	return n.createdAt
}

func (t Type) valid() bool {
	switch t {
	case TypeOrderConfirmed, TypeOrderFailed, TypePaymentFailed, TypeLowStock:
		return true
	default:
		return false
	}
}

func (c Channel) valid() bool {
	switch c {
	case ChannelEmail, ChannelSMS, ChannelPush:
		return true
	default:
		return false
	}
}
