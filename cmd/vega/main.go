package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/go-faster/sdk/zctx"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/go-faster/vega/internal/api"
	"github.com/go-faster/vega/internal/kube"
	"github.com/go-faster/vega/internal/oas"
	"github.com/go-faster/vega/internal/promapi"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, t *app.Telemetry) error {
		ctx = zctx.WithOpenTelemetryZap(ctx)
		kubeClient, err := kube.New(t)
		if err != nil {
			return errors.Wrap(err, "kube.New")
		}
		promURL := os.Getenv("PROMAPI_URL")
		client, err := promapi.NewClient(promURL,
			promapi.WithTracerProvider(t.TracerProvider()),
			promapi.WithMeterProvider(t.MeterProvider()),
			promapi.WithClient(&http.Client{
				Transport: otelhttp.NewTransport(http.DefaultTransport,
					otelhttp.WithMeterProvider(t.MeterProvider()),
					otelhttp.WithTracerProvider(t.TracerProvider()),
					otelhttp.WithPropagators(t.TextMapPropagator()),
					otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
						parsedURL, err := r.URL.Parse(promURL)
						if err != nil {
							return operation
						}
						// Trip prefix from parsed URL.
						trimmedPath := strings.TrimPrefix(r.URL.Path, parsedURL.Path)
						route, ok := (&promapi.Server{}).FindRoute(r.Method, trimmedPath)
						if !ok {
							return operation
						}
						if v := route.Name(); v != "" {
							return "client." + v
						}
						if v := route.PathPattern(); v != "" {
							return v
						}
						return operation
					}),
				),
			}),
		)
		if err != nil {
			return errors.Wrap(err, "create client")
		}
		handler := api.NewHandler(
			kubeClient,
			client,
			t.TracerProvider(),
		)
		srv, err := oas.NewServer(handler)
		if err != nil {
			return errors.Wrap(err, "create server")
		}
		h := &http.Server{
			Addr: ":8080",
			Handler: otelhttp.NewHandler(srv, "",
				otelhttp.WithMeterProvider(t.MeterProvider()),
				otelhttp.WithTracerProvider(t.TracerProvider()),
				otelhttp.WithPropagators(t.TextMapPropagator()),
				otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
					route, ok := srv.FindPath(r.Method, r.URL)
					if !ok {
						return operation
					}
					if v := route.Name(); v != "" {
						return v
					}
					return operation
				}),
			),
			ReadHeaderTimeout: time.Second,
			BaseContext: func(listener net.Listener) context.Context {
				return ctx
			},
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
