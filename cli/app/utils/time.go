// Package utils for miscellaneous functions.
package utils

import "time"

// NanosecondsToMilliseconds for time conversion.
func NanosecondsToMilliseconds(ns int64) int {
	return int(float64(ns) / float64(time.Millisecond))
}
