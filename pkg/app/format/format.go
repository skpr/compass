// Package format renders the values the interface displays.
//
// Every number on screen goes through here, so that the same quantity is
// never written two ways on two pages.
package format

import (
	"fmt"
	"time"
)

// Milliseconds of a duration, truncated.
func Milliseconds(d time.Duration) int64 {
	return d.Milliseconds()
}

// Duration of a span, in the largest unit which keeps it short.
//
// A column of these is scanned rather than read, so the unit changes at the
// point where the digits would otherwise start costing more than they say:
// 8200ms is harder to place at a glance than 8.2s.
func Duration(d time.Duration) string {
	ms := Milliseconds(d)

	switch {
	// A call which ran for less than a millisecond is not a call which took no
	// time. The threshold can be set to zero — the Node compose service does
	// exactly that — so these do reach the screen, and rounding them to 0ms
	// next to a bar which is plainly drawn reads as a contradiction.
	case ms == 0 && d > 0:
		return "<1ms"
	case ms < 1000:
		return fmt.Sprintf("%dms", ms)
	case ms < 60_000:
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	default:
		return fmt.Sprintf("%dm%ds", ms/60_000, (ms%60_000)/1000)
	}
}

// Bytes in a binary unit.
func Bytes(b int64) string {
	const unit = 1024

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// MaxAgePermanent is the max age Drupal uses for "cacheable forever".
const MaxAgePermanent int64 = -1

// MaxAge of a Drupal cacheability value, where permanent is an absence of a
// limit rather than a number worth printing.
func MaxAge(seconds int64) string {
	if seconds == MaxAgePermanent {
		return "permanent"
	}

	return fmt.Sprintf("%ds", seconds)
}

// Percent of a zero-to-one fraction.
func Percent(fraction float64) string {
	return fmt.Sprintf("%.1f%%", fraction*100)
}

// RelativeTime between two instants.
//
// The second argument is the clock rather than time.Now, so that a rendered
// screen is reproducible and can be compared against a golden file.
func RelativeTime(t, now time.Time) string {
	if t.IsZero() {
		return "just now"
	}

	d := now.Sub(t)

	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		return plural(int(d.Seconds()), "s")
	case d < time.Hour:
		return plural(int(d.Minutes()), "m")
	default:
		return plural(int(d.Hours()), "h")
	}
}

// plural of an elapsed count in a unit.
func plural(count int, unit string) string {
	return fmt.Sprintf("%d%s ago", count, unit)
}

// Count of something, with its noun in the right number.
func Count(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}

	return fmt.Sprintf("%d %s", count, plural)
}
