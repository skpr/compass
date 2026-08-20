package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetPathFromProcess_NotRunning(t *testing.T) {
	// A runtime which is not present must be reported as not found, so that the
	// sidecar can skip it instead of failing.
	_, err := GetPathFromProcess(context.Background(), "compass-does-not-exist-4f2b", "/compass.node", 200*time.Millisecond)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetPathFromProcess_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GetPathFromProcess(ctx, "compass-does-not-exist-4f2b", "/compass.node", time.Minute)
	assert.ErrorIs(t, err, context.Canceled)
}
