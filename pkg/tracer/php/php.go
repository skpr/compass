package php

import (
	"context"

	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/php/cli"
	"github.com/skpr/compass/pkg/tracer/php/fpm"
	"github.com/skpr/compass/pkg/tracer/sink"
)

func Run(ctx context.Context, plugin sink.Interface, extensionPath string) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return fpm.Run(ctx, plugin, extensionPath)
	})

	g.Go(func() error {
		return cli.Run(ctx, plugin, extensionPath)
	})

	return g.Wait()
}
