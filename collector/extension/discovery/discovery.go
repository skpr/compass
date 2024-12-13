// Package discovery is used to discover the location of the extension.
package discovery

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/cenkalti/backoff/v4"
	"github.com/shirou/gopsutil/process"
)

// GetPathFromProcess will wait and return the path to the extension for a process.
func GetPathFromProcess(logger *slog.Logger, processName, extensionPath string) (string, error) {
	ticker := backoff.NewTicker(backoff.NewExponentialBackOff())

	for range ticker.C {
		logger.Info("Polling for list of processes")

		pid, ok, err := findParentProcess(processName)
		if err != nil {
			return "", fmt.Errorf("failed to find parent process from list: %w", err)
		}

		if !ok {
			continue
		}

		ticker.Stop()

		path := fmt.Sprintf("/proc/%d/root%s", pid, extensionPath)

		_, err = os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("failed to stat path %s: %w", path, err)
		}

		return path, nil
	}

	return "", fmt.Errorf("timed out")
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
			return 0, false, fmt.Errorf("error getting process name: %w", err)
		}

		if n != name {
			continue
		}

		children, err := p.Children()
		if err != nil {
			continue
		}

		if len(children) > 0 {
			return p.Pid, true, nil
		}
	}

	return 0, false, nil
}
