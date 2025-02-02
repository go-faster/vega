package main

import (
	"context"
	"net/http"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/go-faster/sdk/zctx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/go-faster/vega/internal/api"
	"github.com/go-faster/vega/internal/kube"
	"github.com/go-faster/vega/internal/oas"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, t *app.Telemetry) error {
		ctx = zctx.WithOpenTelemetryZap(ctx)
		kubeClient, err := kube.New(t)
		if err != nil {
			return errors.Wrap(err, "kube.New")
		}
		handler := api.NewHandler(
			kubeClient,
			t.TracerProvider(),
		)
		srv, err := oas.NewServer(handler)
		if err != nil {
			return errors.Wrap(err, "create server")
		}
		h := &http.Server{
			Addr:              ":8080",
			Handler:           srv,
			ReadHeaderTimeout: time.Second,
		}
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			<-ctx.Done()
			return h.Shutdown(ctx)
		})
		g.Go(func() error {
			lg.Info("Server started", zap.String("addr", h.Addr))
			if !errors.Is(h.ListenAndServe(), http.ErrServerClosed) {
				return errors.New("server closed")
			}
			return nil
		})
		return g.Wait()
	},
		app.WithServiceName("vega"),
	)
}
