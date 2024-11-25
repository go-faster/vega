// Binary vega-agent is per-host agent for vega.
package main

import (
	"context"

	"github.com/go-faster/sdk/app"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Metrics) error {
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			<-ctx.Done()
			return ctx.Err()
		})
		return g.Wait()
	})
}
