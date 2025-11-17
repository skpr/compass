// Package time for utility time conversions.
package time

import "time"

// NanosecondsToMilliseconds for time conversion.
func NanosecondsToMilliseconds(ns int64) int {
	return int(float64(ns) / float64(time.Millisecond))
}
