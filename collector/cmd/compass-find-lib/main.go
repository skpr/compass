// Package main provides an entrypoint for the helper script.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector/internal/envget"
)

var (
	cmdLong = `
		Helper script to discovery the parent pid ID by process name. Used by the collector.`

	cmdExample = `
		# Run the helper script and look for PHP FPM's parent process ID.
		compass-find-lib --process-name=php-fpm

		# Run the script with a custom lib path.
		compass-find-lib --process-name=php-fpm --lib-path=/usr/lib/php/modules/something-else.so`
)

func main() {
	var (
		flagProcessName string
		flagLibPath     string
		flagPoll        time.Duration
	)

	cmd := &cobra.Command{
		Use:     "compass-find-lib",
		Short:   "Run the helper script",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

			logger.Info("Script started")

			pid, err := waitForParentProcess(flagProcessName, flagPoll)
			if err != nil {
				return err
			}

			logger.Info(fmt.Sprintf("PID found (%d). Checking if lib exists.", pid))

			path := fmt.Sprintf("/proc/%d/root%s", pid, flagLibPath)

			_, err = os.Stat(path)
			if err != nil {
				return err
			}

			logger.Info("Script has completed")

			// Print the path to stdout for script usage.
			fmt.Println(path)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flagProcessName, "process-name", envget.String("COMPASS_PROCESS_NAME", ""), "Name of the process which will be used for discovery")
	cmd.PersistentFlags().StringVar(&flagLibPath, "lib-path", envget.String("COMPASS_LIB_PATH", "/usr/lib/php/modules/compass.so"), "Path to the Compass extension")
	cmd.PersistentFlags().DurationVar(&flagPoll, "poll", time.Second, "How frequently to poll for current list of processes")

	cmd.Execute()
}

// Helper function to wait for parent process and return the pid.
func waitForParentProcess(name string, duration time.Duration) (int32, error) {
	for {
		time.Sleep(duration)

		processes, err := process.Processes()
		if err != nil {
			return 0, fmt.Errorf("failed to get process list: %w", err)
		}

		pid, ok, err := findParentProcess(processes, name)
		if err != nil {
			return 0, fmt.Errorf("failed to find parent process from list: %w", err)
		}

		if !ok {
			continue
		}

		return pid, nil
	}
}

// Helper function to find the parent process
func findParentProcess(list []*process.Process, name string) (int32, bool, error) {
	for _, p := range list {
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
