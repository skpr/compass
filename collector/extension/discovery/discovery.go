package discovery

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/shirou/gopsutil/process"
)

// GetPathFromProcess will wait and return the path to the extension for a process.
func GetPathFromProcess(logger *slog.Logger, processName, extensionPath string, duration time.Duration) (string, error) {
	for {
		time.Sleep(duration)

		logger.Info("Polling for list of processes")

		processes, err := process.Processes()
		if err != nil {
			return "", fmt.Errorf("failed to get process list: %w", err)
		}

		logger.Info("Looking for parent processes")

		pid, ok, processNames, err := findParentProcess(processes, processName)
		if err != nil {
			return "", fmt.Errorf("failed to find parent process from list: %w", err)
		}

		if !ok {
			logger.Info(fmt.Sprintf("Parent process %s not found in list %s", processName, processNames))
			continue
		}

		path := fmt.Sprintf("/proc/%d/root%s", pid, extensionPath)

		_, err = os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("failed to stat path %s: %w", path, err)
		}

		return path, nil
	}
}

// Helper function to find the parent process
func findParentProcess(list []*process.Process, name string) (int32, bool, []string, error) {
	var names []string

	for _, p := range list {
		n, err := p.Name()
		if err != nil {
			return 0, false, names, fmt.Errorf("error getting process name: %w", err)
		}

		names = append(names, n)

		if n != name {
			continue
		}

		children, err := p.Children()
		if err != nil {
			continue
		}

		if len(children) > 0 {
			return p.Pid, true, names, nil
		}
	}

	return 0, false, names, nil
}
