package node

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/cgroupfilter"
	"github.com/skpr/compass/pkg/tracer/node/http"
	"github.com/skpr/compass/pkg/tracer/sink"
	"github.com/skpr/compass/pkg/tracer/spans"
)

func Run(ctx context.Context, plugin sink.Interface, addonPath string, spanOptions spans.Options, filter cgroupfilter.Filter) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return http.Run(ctx, plugin, addonPath, spanOptions, filter)
	})

	// g.Go(func() error {
	// 	return cli.Run(ctx, plugin, addonPath)
	// })

	return g.Wait()
}
