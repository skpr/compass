package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector/internal/collector"
	"github.com/skpr/compass/collector/plugin"
)

var (
	cmdLong = `
		Run the process which collects tracing data.`

	cmdExample = `
		# Run the collector and target a specific container.
		compass-collector run --container=php-fpm

		# Run the collector and target a different library.
		compass-collector run --lib-path=/usr/lib/php/modules/something-else.so`
)

// NewCommand creates a new cobra.Command for 'alias' sub command
func NewCommand() *cobra.Command {
	var (
		flagPlugin    string
		flagContainer string
		flagLibPath   string
	)

	cmd := &cobra.Command{
		Use:                   "run",
		DisableFlagsInUseLine: true,
		Short:                 "Run the collector",
		Long:                  cmdLong,
		Example:               cmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(context.TODO(), flagContainer, flagLibPath, flagPlugin)
		},
	}

	cmd.PersistentFlags().StringVar(&flagPlugin, "plugin", os.Getenv("COMPASS_COLLECTOR_PLUGIN"), "Plugin for processing tracing data")
	cmd.PersistentFlags().StringVar(&flagContainer, "container", os.Getenv("COMPASS_COLLECTOR_CONTAINER"), "Container to target for collection")
	cmd.PersistentFlags().StringVar(&flagLibPath, "lib-path", "/usr/lib/php/modules/compass.so", "Path to the Compass extension")

	return cmd
}

func run(ctx context.Context, container, libPath, pluginPath string) error {
	// Use the container name to lookup the actual location of the lib.
	if container != "" {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return fmt.Errorf("failed to get Docker client: %w", err)
		}
		defer cli.Close()

		inspect, err := cli.ContainerInspect(ctx, container)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}

		libPath = fmt.Sprintf("/host/proc/%d/root%s", inspect.State.Pid, libPath)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	p, err := plugin.Load(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	return collector.Run(ctx, logger, libPath, p)
}
