// Package discovery is used to discover the location of the extension.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/shirou/gopsutil/process"
)

// ErrNotFound is returned when the process or its extension could not be found.
var ErrNotFound = errors.New("extension not found")

// GetPathFromProcess will wait and return the path to the extension for a process.
//
// ErrNotFound is returned if the process is not running, or is running without
// the extension installed, by the time the timeout is reached. Callers treat
// this as "this runtime is not present" rather than a failure.
func GetPathFromProcess(ctx context.Context, processName, extensionPath string, timeout time.Duration) (string, error) {
	policy := backoff.NewExponentialBackOff()
	policy.MaxElapsedTime = timeout

	ticker := backoff.NewTicker(policy)
	defer ticker.Stop()

	// Reason we could not find the extension, returned when we give up.
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

			path := fmt.Sprintf("/proc/%d/root%s", pid, extensionPath)

			// The process can be running before the extension has been installed,
			// so keep waiting instead of failing outright.
			if _, err := os.Stat(path); err != nil {
				reason = fmt.Errorf("%w: %s", ErrNotFound, err)
				continue
			}

			return path, nil
		}
	}
}

// processInfo is the subset of *process.Process that findMasterProcess needs,
// so master identification can be tested with a mock process list.
type processInfo interface {
	pid() int32
	name() (string, error)
	ppid() (int32, error)
}

type gopsutilProcess struct {
	p *process.Process
}

func (g gopsutilProcess) pid() int32            { return g.p.Pid }
func (g gopsutilProcess) name() (string, error) { return g.p.Name() }
func (g gopsutilProcess) ppid() (int32, error)  { return g.p.Ppid() }

func findParentProcess(name string) (int32, bool, error) {
	processes, err := process.Processes()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get process list: %w", err)
	}

	infos := make([]processInfo, 0, len(processes))
	for _, p := range processes {
		infos = append(infos, gopsutilProcess{p: p})
	}

	return findMasterProcess(name, infos)
}

// findMasterProcess returns the php-fpm process that has no php-fpm parent:
// workers carry the master's PID as their PPID, the master does not. This
// avoids matching "master process" in a command line, which any php-fpm
// invocation containing those words would satisfy.
func findMasterProcess(name string, processes []processInfo) (int32, bool, error) {
	names := make(map[int32]string, len(processes))
	for _, p := range processes {
		n, err := p.name()
		if err != nil {
			continue
		}
		names[p.pid()] = n
	}

	for _, p := range processes {
		if names[p.pid()] != name {
			continue
		}

		ppid, err := p.ppid()
		if err != nil {
			slog.Warn("failed to read parent PID during discovery, skipping process",
				"process", name, "pid", p.pid(), "error", err)
			continue
		}

		if names[ppid] != name {
			return p.pid(), true, nil
		}
	}

	return 0, false, nil
}
