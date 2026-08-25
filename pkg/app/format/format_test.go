package format

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMilliseconds(t *testing.T) {
	assert.Equal(t, int64(1), Milliseconds(time.Millisecond))
	assert.Equal(t, int64(0), Milliseconds(0))
	// Sub-millisecond truncates rather than rounding up, so a span shorter
	// than the probe threshold never reports as a whole millisecond.
	assert.Equal(t, int64(0), Milliseconds(500*time.Microsecond))
	assert.Equal(t, int64(5000), Milliseconds(5*time.Second))
}

func TestDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{name: "zero", duration: 0, expected: "0ms"},
		{name: "sub millisecond", duration: 40 * time.Microsecond, expected: "<1ms"},
		{name: "just under a millisecond", duration: time.Millisecond - 1, expected: "<1ms"},
		{name: "milliseconds", duration: 402 * time.Millisecond, expected: "402ms"},
		{name: "just under a second", duration: 999 * time.Millisecond, expected: "999ms"},
		{name: "one second", duration: time.Second, expected: "1.0s"},
		{name: "seconds", duration: 8200 * time.Millisecond, expected: "8.2s"},
		{name: "just under a minute", duration: 59900 * time.Millisecond, expected: "59.9s"},
		{name: "minutes", duration: 62 * time.Second, expected: "1m2s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Duration(tt.duration))
		})
	}
}

func TestBytes(t *testing.T) {
	assert.Equal(t, "500 B", Bytes(500))
	assert.Equal(t, "0 B", Bytes(0))
	assert.Equal(t, "1.0 KB", Bytes(1024))
	assert.Equal(t, "1.0 MB", Bytes(1024*1024))
	assert.Equal(t, "1.0 GB", Bytes(1024*1024*1024))
}

func TestMaxAge(t *testing.T) {
	assert.Equal(t, "permanent", MaxAge(MaxAgePermanent))
	assert.Equal(t, "0s", MaxAge(0))
	assert.Equal(t, "3600s", MaxAge(3600))
}

func TestPercent(t *testing.T) {
	assert.Equal(t, "0.0%", Percent(0))
	assert.Equal(t, "40.9%", Percent(0.409))
	assert.Equal(t, "100.0%", Percent(1))
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		at       time.Time
		expected string
	}{
		{name: "zero value", at: time.Time{}, expected: "just now"},
		{name: "sub second", at: now.Add(-500 * time.Millisecond), expected: "just now"},
		{name: "one second", at: now.Add(-time.Second), expected: "1s ago"},
		{name: "seconds", at: now.Add(-42 * time.Second), expected: "42s ago"},
		{name: "one minute", at: now.Add(-time.Minute), expected: "1m ago"},
		{name: "minutes", at: now.Add(-9 * time.Minute), expected: "9m ago"},
		{name: "hours", at: now.Add(-3 * time.Hour), expected: "3h ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, RelativeTime(tt.at, now))
		})
	}
}

func TestCount(t *testing.T) {
	assert.Equal(t, "0 callers", Count(0, "caller", "callers"))
	assert.Equal(t, "1 caller", Count(1, "caller", "callers"))
	assert.Equal(t, "2 callers", Count(2, "caller", "callers"))
}
