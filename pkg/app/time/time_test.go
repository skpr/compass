package time

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNanosecondsToMilliseconds(t *testing.T) {
	assert.Equal(t, 1, NanosecondsToMilliseconds(1_000_000))
}

func TestNanosecondsToMilliseconds_Zero(t *testing.T) {
	assert.Equal(t, 0, NanosecondsToMilliseconds(0))
}

func TestNanosecondsToMilliseconds_SubMillisecond(t *testing.T) {
	// 500,000 ns = 0.5 ms, truncated to 0.
	assert.Equal(t, 0, NanosecondsToMilliseconds(500_000))
}

func TestNanosecondsToMilliseconds_LargeValue(t *testing.T) {
	// 5 seconds = 5,000,000,000 ns = 5000 ms.
	assert.Equal(t, 5000, NanosecondsToMilliseconds(5_000_000_000))
}
