package clock_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/infrastructure/clock"
)

func TestSystemClockNow(t *testing.T) {
	c := clock.New()
	now := c.Now()
	assert.WithinDuration(t, time.Now(), now, time.Minute)
	assert.Equal(t, time.UTC, now.Location())
}
