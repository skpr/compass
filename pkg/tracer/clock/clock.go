// Package clock turns the timestamps the probes emit into instants.
//
// bpf_ktime_get_ns() counts nanoseconds since the machine booted. That is the
// right clock to measure a request with, because nothing can step it out from
// under a running trace the way NTP can step the wall clock, and the wrong one
// to record: a count since boot means nothing on its own, and nothing outside
// this machine can be lined up against it.
//
// So the offset between the two clocks is read once, at startup, and added to
// every timestamp. What a trace measures stays monotonic, and what it carries
// is a real instant.
package clock

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// Monotonic relates the monotonic clock the probes read to the wall clock.
type Monotonic struct {
	// Boot is the wall clock instant at which the monotonic clock read zero.
	Boot time.Time
}

// System reads the offset between the two clocks.
//
// The two are not read at the same instant, so the offset carries the time
// between the calls as error. That is a scheduling quantum at worst, against
// timestamps which are only ever compared within a single request.
func System() (Monotonic, error) {
	var ts unix.Timespec

	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return Monotonic{}, fmt.Errorf("failed to read the monotonic clock: %w", err)
	}

	return Monotonic{Boot: time.Now().Add(-time.Duration(ts.Nano()))}, nil
}

// Time of a timestamp taken from the monotonic clock.
func (m Monotonic) Time(ns uint64) time.Time {
	return m.Boot.Add(time.Duration(ns))
}
