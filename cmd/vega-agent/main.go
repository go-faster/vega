// Binary vega-agent is per-host agent for vega.
package main

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/cilium/cilium/api/v1/observer"
	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/go-faster/tetragon/api/v1/tetragon"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Telemetry) error {
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			<-ctx.Done()
			return ctx.Err()
		})
		meter := m.MeterProvider().Meter("vega-agent")
		g.Go(func() error {
			// Hubble component.
			const (
				hubblePath   = "/var/run/cilium/hubble.sock"
				hubbleTarget = "unix://" + hubblePath
			)
			{
				_, err := os.Stat(hubblePath)
				if err != nil {
					return errors.Wrap(err, "tetragon socket")
				}
			}
			hubbleConn, err := grpc.NewClient(hubbleTarget,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				return errors.Wrap(err, "tetragon grpc")
			}
			client := observer.NewObserverClient(hubbleConn)
			{
				ctx, cancel := context.WithTimeout(ctx, time.Second*5)
				defer cancel()

				res, err := client.ServerStatus(ctx, &observer.ServerStatusRequest{})
				if err != nil {
					return errors.Wrap(err, "server status")
				}
				lg.Info("hubble version", zap.String("version", res.Version))
			}
			b, err := client.GetFlows(ctx, &observer.GetFlowsRequest{
				Follow: true,
			})
			if err != nil {
				return errors.Wrap(err, "get flows")
			}
			logger := lg.Named("vega-agent.flows")
			flowsCount, err := meter.Int64Counter("agent.hubble.flows_count", metric.WithDescription("Number of received flows"))
			if err != nil {
				return errors.Wrap(err, "create counter")
			}
			for {
				resp, err := b.Recv()
				switch {
				case errors.Is(err, io.EOF), errors.Is(err, context.Canceled):
					return nil
				case err == nil:
				default:
					if status.Code(err) == codes.Canceled {
						return nil
					}
					return errors.Wrap(err, "recv")
				}
				logger.Info("Got flow",
					zap.String("node", resp.NodeName),
				)
				flowsCount.Add(ctx, 1)
			}
		})
		g.Go(func() error {
			// Tetragon component.
			const (
				tetragonPath   = "/var/run/tetragon/tetragon.sock"
				tetragonTarget = "unix://" + tetragonPath
			)
			{
				_, err := os.Stat(tetragonPath)
				if err != nil {
					return errors.Wrap(err, "tetragon socket")
				}
			}
			tetragonConn, err := grpc.NewClient(tetragonTarget,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)
			if err != nil {
				return errors.Wrap(err, "tetragon grpc")
			}
			client := tetragon.NewFineGuidanceSensorsClient(tetragonConn)
			{
				ctx, cancel := context.WithTimeout(ctx, time.Second*5)
				defer cancel()

				version, err := client.GetVersion(ctx, &tetragon.GetVersionRequest{})
				if err != nil {
					return errors.Wrap(err, "get version")
				}
				lg.Info("tetragon version", zap.String("version", version.Version))
			}
			logger := lg.Named("vega-agent.tetragon.events")
			eventsCount, err := meter.Int64Counter("agent.tetragon.events_count", metric.WithDescription("Number of received events"))
			if err != nil {
				return errors.Wrap(err, "create counter")
			}
			b, err := client.GetEvents(ctx, &tetragon.GetEventsRequest{})
			if err != nil {
				return errors.Wrap(err, "get events")
			}
			for {
				resp, err := b.Recv()
				switch {
				case errors.Is(err, io.EOF), errors.Is(err, context.Canceled):
					return nil
				case err == nil:
				default:
					if status.Code(err) == codes.Canceled {
						return nil
					}
					return errors.Wrap(err, "recv")
				}
				logger.Info("Got event",
					zap.String("node", resp.NodeName),
				)
				eventsCount.Add(ctx, 1)
			}
		})
		return g.Wait()
	},
		app.WithResource(func(ctx context.Context) (*resource.Resource, error) {
			return resource.New(ctx,
				resource.WithOS(),
				resource.WithFromEnv(),
				resource.WithTelemetrySDK(),
				resource.WithHost(),
				resource.WithProcess(),
				resource.WithAttributes(
					attribute.String("service.name", "vega.agent"),
				),
			)
		}),
	)
}
