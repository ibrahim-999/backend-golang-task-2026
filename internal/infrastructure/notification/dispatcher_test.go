package notificationinfra_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
	domnotification "github.com/ibrahim-999/backend-golang-task-2026/internal/domain/notification"
	notificationinfra "github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/notification"
)

type fakeNotifRepo struct {
	mu      sync.Mutex
	created int
	updated int
	nextID  uint64
}

func (r *fakeNotifRepo) Create(ctx context.Context, n *domnotification.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	n.AssignID(r.nextID)
	r.created++
	return nil
}

func (r *fakeNotifRepo) Update(ctx context.Context, n *domnotification.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updated++
	return nil
}

func (r *fakeNotifRepo) ListByStatus(ctx context.Context, status domnotification.Status, limit int) ([]*domnotification.Notification, error) {
	return nil, nil
}

func (r *fakeNotifRepo) updates() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.updated
}

type fakeSender struct {
	mu   sync.Mutex
	sent int
	fail bool
}

func (s *fakeSender) Send(ctx context.Context, n *domnotification.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fail {
		return errors.New("send failed")
	}
	s.sent++
	return nil
}

func (s *fakeSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent
}

func newNotif(t *testing.T) *domnotification.Notification {
	t.Helper()
	uid := uint64(7)
	n, err := domnotification.NewNotification(&uid, domnotification.TypeOrderConfirmed, domnotification.ChannelEmail, "Subject", "body")
	require.NoError(t, err)
	return n
}

func TestDispatcherSendsAndMarksSent(t *testing.T) {
	repo := &fakeNotifRepo{}
	sender := &fakeSender{}
	d := notificationinfra.NewDispatcher(repo, sender, 2, 8, zerolog.Nop())
	d.Start()
	require.NoError(t, d.Enqueue(context.Background(), newNotif(t)))
	require.Eventually(t, func() bool { return sender.count() >= 1 && repo.updates() >= 1 }, 2*time.Second, 10*time.Millisecond)
	d.Stop()
}

func TestDispatcherSendFailureMarksFailed(t *testing.T) {
	repo := &fakeNotifRepo{}
	sender := &fakeSender{fail: true}
	d := notificationinfra.NewDispatcher(repo, sender, 1, 4, zerolog.Nop())
	d.Start()
	require.NoError(t, d.Enqueue(context.Background(), newNotif(t)))
	require.Eventually(t, func() bool { return repo.updates() >= 1 }, 2*time.Second, 10*time.Millisecond)
	d.Stop()
}

func TestLogSender(t *testing.T) {
	s := notificationinfra.NewLogSender(zerolog.Nop())
	require.NoError(t, s.Send(context.Background(), newNotif(t)))
}

var _ ports.NotificationRepository = (*fakeNotifRepo)(nil)
