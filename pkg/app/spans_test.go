package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes_Bytes(t *testing.T) {
	assert.Equal(t, "500 B", formatBytes(500))
}

func TestFormatBytes_Zero(t *testing.T) {
	assert.Equal(t, "0 B", formatBytes(0))
}

func TestFormatBytes_KB(t *testing.T) {
	assert.Equal(t, "1.0 KB", formatBytes(1024))
}

func TestFormatBytes_MB(t *testing.T) {
	assert.Equal(t, "1.0 MB", formatBytes(1024*1024))
}

func TestFormatBytes_GB(t *testing.T) {
	assert.Equal(t, "1.0 GB", formatBytes(1024*1024*1024))
}
