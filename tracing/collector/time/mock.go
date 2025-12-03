package time

import "time"

// Mock for testing.
type Mock struct {
	now time.Time
}

// NewMock for testing.
func NewMock(now time.Time) *Mock {
	return &Mock{
		now: now,
	}
}

// Now returns the mock current time.
func (m *Mock) Now() time.Time {
	return m.now
}
