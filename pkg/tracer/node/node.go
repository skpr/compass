package node

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/node/http"
	"github.com/skpr/compass/pkg/tracer/sink"
)

func Run(ctx context.Context, plugin sink.Interface, addonPath string, maxFunctionCalls int) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return http.Run(ctx, plugin, addonPath, maxFunctionCalls)
	})

	// g.Go(func() error {
	// 	return cli.Run(ctx, plugin, addonPath)
	// })

	return g.Wait()
}
