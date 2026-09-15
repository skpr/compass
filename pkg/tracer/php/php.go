package php

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/cgroupfilter"
	"github.com/skpr/compass/pkg/tracer/php/cli"
	"github.com/skpr/compass/pkg/tracer/php/fpm"
	"github.com/skpr/compass/pkg/tracer/sink"
	"github.com/skpr/compass/pkg/tracer/spans"
)

func Run(ctx context.Context, plugin sink.Interface, extensionPath string, spanOptions spans.Options, filter cgroupfilter.Filter) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return fpm.Run(ctx, plugin, extensionPath, spanOptions, filter)
	})

	g.Go(func() error {
		return cli.Run(ctx, plugin, extensionPath, spanOptions, filter)
	})

	return g.Wait()
}
