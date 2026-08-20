// Package tracer implements the collection of telemetry from instrumented runtimes.
package tracer

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/node"
	"github.com/skpr/compass/pkg/tracer/php"
	"github.com/skpr/compass/pkg/tracer/sink"
)

// Runtimes which have been discovered and can be traced.
//
// A deployment usually only runs one of these, so an empty path means that
// runtime is not present and will not be traced.
type Runtimes struct {
	// PHPExtensionPath is the path to the Compass PHP extension.
	PHPExtensionPath string
	// NodeAddonPath is the path to the Compass Node addon.
	NodeAddonPath string
}

// Empty returns true when there is nothing to trace.
func (r Runtimes) Empty() bool {
	return r.PHPExtensionPath == "" && r.NodeAddonPath == ""
}

// Run the collector for all discovered runtimes.
func Run(ctx context.Context, plugin sink.Interface, runtimes Runtimes) error {
	if runtimes.Empty() {
		return fmt.Errorf("no runtimes to trace")
	}

	g, ctx := errgroup.WithContext(ctx)

	if runtimes.PHPExtensionPath != "" {
		g.Go(func() error {
			return php.Run(ctx, plugin, runtimes.PHPExtensionPath)
		})
	}

	if runtimes.NodeAddonPath != "" {
		g.Go(func() error {
			return node.Run(ctx, plugin, runtimes.NodeAddonPath)
		})
	}

	return g.Wait()
}
