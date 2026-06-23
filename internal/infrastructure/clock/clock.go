package clock

import (
	"time"

	"github.com/ibrahim-999/backend-golang-task-2026/internal/application/ports"
)

type System struct{}

func New() System {
	return System{}
}

func (System) Now() time.Time {
	return time.Now().UTC()
}

var _ ports.Clock = System{}
