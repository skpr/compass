// Package discovery is used to discover the location of the addon.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/shirou/gopsutil/process"
)

// ErrNotFound is returned when the process or its addon could not be found.
var ErrNotFound = errors.New("addon not found")

// GetPathFromProcess will wait and return the path to the addon for a process.
//
// ErrNotFound is returned if the process is not running, or is running without
// the addon installed, by the time the timeout is reached. Callers treat
// this as "this runtime is not present" rather than a failure.
func GetPathFromProcess(ctx context.Context, processName, addonPath string, timeout time.Duration) (string, error) {
	policy := backoff.NewExponentialBackOff()
	policy.MaxElapsedTime = timeout

	ticker := backoff.NewTicker(policy)
	defer ticker.Stop()

	// Reason we could not find the addon, returned when we give up.
	reason := ErrNotFound

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case _, ok := <-ticker.C:
			if !ok {
				// The backoff policy has been exhausted.
				return "", reason
			}

			pid, found, err := findParentProcess(processName)
			if err != nil {
				return "", fmt.Errorf("failed to find parent process from list: %w", err)
			}

			if !found {
				reason = fmt.Errorf("%w: process not running: %s", ErrNotFound, processName)
				continue
			}

			path := fmt.Sprintf("/proc/%d/root%s", pid, addonPath)

			// The process can be running before the addon has been installed,
			// so keep waiting instead of failing outright.
			if _, err := os.Stat(path); err != nil {
				reason = fmt.Errorf("%w: %s", ErrNotFound, err)
				continue
			}

			return path, nil
		}
	}
}

// Helper function to find the parent process
func findParentProcess(name string) (int32, bool, error) {
	processes, err := process.Processes()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get process list: %w", err)
	}

	for _, p := range processes {
		n, err := p.Name()
		if err != nil {
			// The process may have exited while we were looking at it.
			continue
		}

		if n != name {
			continue
		}

		// Node has no master/worker split, so the first match is the
		// process to attach to; no PPID disambiguation is needed here.
		return p.Pid, true, nil
	}

	return 0, false, nil
}
