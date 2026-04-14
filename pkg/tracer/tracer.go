package tracer

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/php"
	"github.com/skpr/compass/pkg/tracer/node"
	"github.com/skpr/compass/pkg/tracer/sink"
)

func Run(ctx context.Context, plugin sink.Interface, phpExtensionPath, nodeAddonPath string) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return php.Run(ctx, plugin, phpExtensionPath)
	})

	g.Go(func() error {
		return node.Run(ctx, plugin, nodeAddonPath)
	})

	return g.Wait()
}
