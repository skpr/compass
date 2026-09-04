package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockProcess is a test stand-in for a real process.
type mockProcess struct {
	mockPid     int32
	mockName    string
	mockNameErr error
	mockPpid    int32
	mockPpidErr error
}

func (m mockProcess) pid() int32            { return m.mockPid }
func (m mockProcess) name() (string, error) { return m.mockName, m.mockNameErr }
func (m mockProcess) ppid() (int32, error)  { return m.mockPpid, m.mockPpidErr }

func TestGetPathFromProcess_NotRunning(t *testing.T) {
	// A runtime which is not present must be reported as not found, so that the
	// sidecar can skip it instead of failing.
	_, err := GetPathFromProcess(context.Background(), "compass-does-not-exist-4f2b", "/compass.so", 200*time.Millisecond)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetPathFromProcess_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GetPathFromProcess(ctx, "compass-does-not-exist-4f2b", "/compass.so", time.Minute)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFindMasterProcess(t *testing.T) {
	const name = "php-fpm"

	tests := []struct {
		name      string
		processes []processInfo
		wantPid   int32
		wantFound bool
	}{
		{
			name: "master with init parent and workers",
			processes: []processInfo{
				mockProcess{mockPid: 10, mockName: name, mockPpid: 1},
				mockProcess{mockPid: 11, mockName: name, mockPpid: 10},
				mockProcess{mockPid: 12, mockName: name, mockPpid: 10},
			},
			wantPid:   10,
			wantFound: true,
		},
		{
			name: "master parent absent from list",
			processes: []processInfo{
				mockProcess{mockPid: 20, mockName: name, mockPpid: 99},
				mockProcess{mockPid: 21, mockName: name, mockPpid: 20},
			},
			wantPid:   20,
			wantFound: true,
		},
		{
			name: "worker whose master parent is also present is not chosen",
			processes: []processInfo{
				mockProcess{mockPid: 30, mockName: name, mockPpid: 1},
				mockProcess{mockPid: 31, mockName: name, mockPpid: 30},
			},
			wantPid:   30,
			wantFound: true,
		},
		{
			name: "cmdline containing 'master process' is no longer a false positive",
			processes: []processInfo{
				mockProcess{mockPid: 40, mockName: name, mockPpid: 1},
				mockProcess{mockPid: 41, mockName: name, mockPpid: 40},
			},
			wantPid:   40,
			wantFound: true,
		},
		{
			name: "no php-fpm process",
			processes: []processInfo{
				mockProcess{mockPid: 50, mockName: "nginx", mockPpid: 1},
				mockProcess{mockPid: 51, mockName: "node", mockPpid: 1},
			},
			wantFound: false,
		},
		{
			name: "process whose ppid errors is skipped, master still found",
			processes: []processInfo{
				mockProcess{mockPid: 60, mockName: name, mockPpidErr: errors.New("permission denied")},
				mockProcess{mockPid: 61, mockName: name, mockPpid: 1},
				mockProcess{mockPid: 62, mockName: name, mockPpid: 61},
			},
			wantPid:   61,
			wantFound: true,
		},
		{
			name: "process whose name errors is ignored",
			processes: []processInfo{
				mockProcess{mockPid: 70, mockNameErr: errors.New("gone")},
				mockProcess{mockPid: 71, mockName: name, mockPpid: 1},
			},
			wantPid:   71,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pid, found, err := findMasterProcess(name, tt.processes)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantPid, pid)
			}
		})
	}
}
