package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cilium/cilium/api/v1/observer"
	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/go-faster/sdk/autometric"
	"github.com/go-faster/sdk/otelsync"
	"github.com/go-faster/tetragon/api/v1/tetragon"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/go-faster/vega/internal/flow"
	"github.com/go-faster/vega/internal/sec"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Telemetry) (err error) {
		a, err := NewApp(lg, m)
		if err != nil {
			return errors.Wrap(err, "init")
		}
		return a.Run(ctx)
	}, app.WithServiceName("vega.ingest"))
}

type App struct {
	log       *zap.Logger
	telemetry *app.Telemetry
	servers   []Server
	metrics   Metrics
	ingesters []EntriesIngester
}

type Server struct {
	Addr     string
	DB       string
	User     string
	Password string
}

func NewApp(lg *zap.Logger, telemetry *app.Telemetry) (*App, error) {
	var servers []Server
	lg.Info("Using config from env")
	for _, addr := range strings.Split(os.Getenv("CLICKHOUSE_ADDR"), ",") {
		servers = append(servers, Server{
			Addr:     addr,
			DB:       os.Getenv("CLICKHOUSE_DB"),
			User:     os.Getenv("CLICKHOUSE_USER"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		})
	}

	a := &App{
		log:       lg,
		telemetry: telemetry,
		servers:   servers,
	}
	lg.Info("Configured",
		zap.Int("servers", len(servers)),
	)
	meter := telemetry.MeterProvider().Meter("")
	adapter := otelsync.NewAdapter(meter)
	var err error
	if a.metrics.OffsetCommited, err = adapter.GaugeInt64("vega.ingest.offset.commited"); err != nil {
		return nil, errors.Wrap(err, "metric adapter gauge")
	}
	if a.metrics.OffsetRead, err = adapter.GaugeInt64("vega.ingest.offset.read"); err != nil {
		return nil, errors.Wrap(err, "metric adapter gauge")
	}
	if err := autometric.Init(meter, &a.metrics, autometric.InitOptions{Prefix: "vega.ingest."}); err != nil {
		return nil, errors.Wrap(err, "autometric")
	}
	if _, err := adapter.Register(); err != nil {
		return nil, errors.Wrap(err, "metric adapter register")
	}

	a.initIngesters()

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	if err := a.setup(ctx); err != nil {
		return errors.Wrap(err, "setup")
	}
	g, ctx := errgroup.WithContext(ctx)
	a.consume(ctx, g)
	a.ingest(ctx, g)
	return g.Wait()
}

type EntriesIngester interface {
	Ingest(ctx context.Context) error
	Consume(ctx context.Context) error
	Setup(ctx context.Context) error
}

func (a *App) setup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	for _, ingester := range a.ingesters {
		if err := ingester.Setup(ctx); err != nil {
			return errors.Wrap(err, "ingester setup")
		}
	}

	return nil
}

func (a *App) consume(ctx context.Context, g *errgroup.Group) {
	for _, ingester := range a.ingesters {
		g.Go(func() error {
			return ingester.Consume(ctx)
		})
	}
}

func (a *App) ingest(ctx context.Context, g *errgroup.Group) {
	for _, ingester := range a.ingesters {
		g.Go(func() error {
			return ingester.Ingest(ctx)
		})
	}
}

func (a *App) initIngesters() {
	const (
		tetragonName = "tetragon"
		hubbleName   = "hubble"
	)
	a.ingesters = append(a.ingesters,
		NewIngester[*tetragon.GetEventsResponse, *sec.Table](IngesterOptions[*tetragon.GetEventsResponse, *sec.Table]{
			Metrics:   a.metrics,
			Telemetry: a.telemetry,
			Servers:   a.servers,
			TableName: tetragonName,
			Group:     "vega.ingest." + tetragonName,
			Topic:     tetragonName,
			DDL:       sec.NewDDL(tetragonName),
			NewTable:  sec.NewTable,
			AppendEntry: func(t *sec.Table, e *Entry[*tetragon.GetEventsResponse]) error {
				return t.Append(sec.Row{Res: e.Res})
			},
			NewMessage: func() *tetragon.GetEventsResponse {
				return &tetragon.GetEventsResponse{}
			},
			Log: a.log.With(zap.String("ingester", tetragonName)),
		}),
		NewIngester[*observer.GetFlowsResponse, *flow.Table](IngesterOptions[*observer.GetFlowsResponse, *flow.Table]{
			Metrics:   a.metrics,
			Telemetry: a.telemetry,
			Servers:   a.servers,
			TableName: hubbleName,
			Group:     "vega.ingest." + hubbleName,
			Topic:     hubbleName,
			DDL:       flow.NewDDL(hubbleName),
			NewTable:  flow.NewTable,
			AppendEntry: func(t *flow.Table, e *Entry[*observer.GetFlowsResponse]) error {
				f := e.Res.GetFlow()
				if f == nil {
					// Skip.
					return nil
				}

				index := flow.Peer{
					Kubernetes: flow.RowKubernetes{
						Namespace: f.GetSource().GetNamespace(),
						Pod:       f.GetSource().GetPodName(),
					},
				}
				peer := flow.Peer{
					Kubernetes: flow.RowKubernetes{
						Namespace: f.GetDestination().GetNamespace(),
						Pod:       f.GetDestination().GetPodName(),
					},
				}

				if err := t.Append(flow.Row{
					Raw:   f,
					Index: index,
					Peer:  peer,
				}); err != nil {
					return errors.Wrap(err, "append index")
				}
				if err := t.Append(flow.Row{
					Raw:     f,
					Index:   peer,
					Peer:    index,
					Inverse: true,
				}); err != nil {
					return errors.Wrap(err, "append inverse")
				}

				return nil
			},
			NewMessage: func() *observer.GetFlowsResponse {
				return &observer.GetFlowsResponse{}
			},
			Log: a.log.With(zap.String("ingester", hubbleName)),
		}),
	)
}
