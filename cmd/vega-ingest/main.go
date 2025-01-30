package main

import (
	"context"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/go-faster/sdk/otelsync"
	"github.com/go-faster/tetragon/api/v1/tetragon"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/go-faster/vega/internal/kfk"
	"github.com/go-faster/vega/internal/sec"
)

type Entry struct {
	Raw     []byte
	Message kafka.Message
	Res     *tetragon.GetEventsResponse
}

func (e *Entry) traceAttributes() []attribute.KeyValue {
	var out []attribute.KeyValue
	// TODO: add some attributes
	return out
}

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Telemetry) (err error) {
		a, err := NewApp(lg, m)
		if err != nil {
			return errors.Wrap(err, "init")
		}
		return a.Run(ctx)
	})
}

type App struct {
	log     *zap.Logger
	metrics *app.Telemetry

	sec    chan *Entry
	reader atomic.Pointer[kafka.Reader]

	initializeDB bool
	servers      []Server

	parseCount  metric.Int64Counter
	parseErrors metric.Int64Counter

	secRead  metric.Int64Counter
	secSaved metric.Int64Counter

	secOffsetRead      metric.Int64Observer
	secOffsetCommitted metric.Int64Observer
}

type Server struct {
	Addr     string
	Table    string
	DB       string
	User     string
	Password string
}

func NewApp(lg *zap.Logger, telemetry *app.Telemetry) (*App, error) {
	var servers []Server
	lg.Info("Using config from env")
	tableName := os.Getenv("CLICKHOUSE_TABLE")
	if tableName == "" {
		tableName = "sec"
	}
	for _, addr := range strings.Split(os.Getenv("CLICKHOUSE_ADDR"), ",") {
		servers = append(servers, Server{
			Addr:     addr,
			Table:    tableName,
			DB:       os.Getenv("CLICKHOUSE_DB"),
			User:     os.Getenv("CLICKHOUSE_USER"),
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		})
	}

	a := &App{
		log:     lg,
		metrics: telemetry,
		sec:     make(chan *Entry),

		servers:      servers,
		initializeDB: true,
	}

	lg.Info("Configured",
		zap.Int("servers", len(servers)),
	)

	meter := telemetry.MeterProvider().Meter("")
	adapter := otelsync.NewAdapter(meter)

	var err error
	if a.secSaved, err = meter.Int64Counter("sec.entries.saved_count"); err != nil {
		return nil, err
	}
	if a.secRead, err = meter.Int64Counter("sec.entries.read_count"); err != nil {
		return nil, err
	}
	if a.parseErrors, err = meter.Int64Counter("sec.parse.errors_count"); err != nil {
		return nil, err
	}
	if a.parseCount, err = meter.Int64Counter("sec.parse.count"); err != nil {
		return nil, err
	}
	if a.secOffsetRead, err = adapter.GaugeInt64("sec.entries.kafka.offset.read"); err != nil {
		return nil, err
	}
	if a.secOffsetCommitted, err = adapter.GaugeInt64("sec.entries.kafka.offset.committed"); err != nil {
		return nil, err
	}

	if _, err := adapter.Register(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.initializeDB {
		if err := a.setup(ctx); err != nil {
			return errors.Wrap(err, "setup")
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return a.consume(ctx) })
	g.Go(func() error { return a.ingest(ctx) })
	return g.Wait()
}

func clickHouseServer(list []Server) Server {
	return list[rand.Intn(len(list))] // #nosec G404
}

func (a *App) setupClickHouse(ctx context.Context, s Server) error {
	a.log.Info("Setting up ClickHouse",
		zap.String("addr", s.Addr),
		zap.String("db", s.DB),
		zap.String("user", s.User),
		zap.String("logs_table", s.Table),
	)
	db, err := ch.Dial(ctx, ch.Options{
		Address:  s.Addr,
		User:     s.User,
		Password: s.Password,
		Database: s.DB,
		Logger:   a.log.Named("ch"),

		OpenTelemetryInstrumentation: true,

		MeterProvider:  a.metrics.MeterProvider(),
		TracerProvider: a.metrics.TracerProvider(),
	})
	if err != nil {
		return errors.Wrap(err, "clickhouse")
	}
	defer func() {
		_ = db.Close()
	}()
	if err := db.Ping(ctx); err != nil {
		return errors.Wrap(err, "clickhouse ping")
	}
	a.log.Info("Connected to clickhouse")
	ddl := sec.NewDDL(s.Table)
	ddl += "\nTTL toDateTime(timestamp) + INTERVAL 6 HOUR"
	if err := db.Do(ctx, ch.Query{Body: ddl}); err != nil {
		return errors.Wrap(err, "log ddl")
	}

	return nil
}

func (a *App) setup(ctx context.Context) error {
	for _, server := range a.servers {
		if err := a.setupClickHouse(ctx, server); err != nil {
			return errors.Wrapf(err, "setup clickhouse traces %s", server.Addr)
		}
	}
	return nil
}

const (
	kafkaMinSizeBytes = 1014 * 25        // 25kb
	kafkaMaxSizeBytes = 1024 * 1024 * 10 // 10 mb
	kafkaMaxWait      = 3 * time.Second
)

func (a *App) consume(ctx context.Context) error {
	const group = "vega.ingest"
	lg := a.log.Named("sec")
	readerConfig := kafka.ReaderConfig{
		GroupTopics: []string{
			"tetragon",
		},
		Brokers:  kfk.Addrs(),
		Dialer:   kfk.Dialer(),
		GroupID:  group,
		MinBytes: kafkaMinSizeBytes,
		MaxBytes: kafkaMaxSizeBytes,
		MaxWait:  kafkaMaxWait,
		Logger: fnLogger(func(s string, i ...interface{}) {
			lg.Sugar().Debugf(s, i...)
		}),
		ErrorLogger: fnLogger(func(s string, i ...interface{}) {
			lg.Sugar().Errorf(s, i...)
		}),
	}
	r := kafka.NewReader(readerConfig)
	a.reader.Store(r)
	defer func() {
		if err := a.reader.Load().Close(); err != nil {
			lg.Error("Close kafka reader", zap.Error(err))
		}
	}()

	for {
		msg, err := r.FetchMessage(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return errors.Wrap(err, "next")
		}
		e, err := a.entry(msg)
		if err != nil {
			return errors.Wrap(err, "flow entry parse")
		}
		a.secRead.Add(ctx, 1, metric.WithAttributes(e.traceAttributes()...))
		a.secOffsetRead.Observe(msg.Offset, metric.WithAttributes(kafkaAttributes(msg, a.reader.Load().Stats())...))
		select {
		case a.sec <- e:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type fnLogger func(string, ...interface{})

func (f fnLogger) Printf(s string, i ...interface{}) {
	f(s, i...)
}

// Copy byte slice.
func Copy(v []byte) []byte {
	b := make([]byte, len(v))
	copy(b, v)
	return b
}

func (a *App) entry(msg kafka.Message) (*Entry, error) {
	var f tetragon.GetEventsResponse

	if err := proto.Unmarshal(msg.Value, &f); err != nil {
		return nil, errors.Wrap(err, "unmarshal sec")
	}

	e := &Entry{
		Raw: Copy(msg.Value),
		Res: &f,
	}

	return e, nil
}

func appendEntry(t *sec.Table, e *Entry) error {
	if err := t.Append(sec.Row{
		Res: e.Res,
	}); err != nil {
		return errors.Wrap(err, "append")
	}

	return nil
}

const (
	// ingestHardTimeout is limit for INSERT query stream duration.
	//
	// When limit is reached, we create new INSERT query stream.
	ingestHardTimeout = time.Second * 15
	// ingestSoftTimeout is time we wait for single batch (data block) to buffer.
	ingestSoftTimeout = time.Millisecond * 300
	// ingestMaxBatch is maximum number of rows in single batch (data block) to buffer.
	ingestMaxBatch = 10_000
)

func (a *App) ingest(ctx context.Context) error {
	hardTicker := time.NewTicker(ingestHardTimeout)
	defer hardTicker.Stop()
	softTicker := time.NewTicker(ingestSoftTimeout)
	defer softTicker.Stop()

	for {
		s := clickHouseServer(a.servers)
		t := sec.NewTable(s.Table)

		db, err := ch.Dial(ctx, ch.Options{
			Logger:      a.log.Named("sec"),
			Address:     s.Addr,
			User:        s.User,
			Password:    s.Password,
			Database:    s.DB,
			Compression: ch.CompressionLZ4,

			OpenTelemetryInstrumentation: true,

			MeterProvider:  a.metrics.MeterProvider(),
			TracerProvider: a.metrics.TracerProvider(),
		})
		if err != nil {
			return errors.Wrap(err, "clickhouse")
		}

		var latest kafka.Message
		if err := db.Do(ctx, ch.Query{
			Body:  t.Insert(),
			Input: t.Input(),
			OnInput: func(ctx context.Context) error {
				t.Reset()
				for {
					if t.Rows() > ingestMaxBatch {
						// Finish batch.
						a.secSaved.Add(ctx, int64(t.Rows()))
						return nil
					}
					select {
					case e := <-a.sec:
						if err := appendEntry(t, e); err != nil {
							a.log.Warn("Append entry",
								zap.Error(err),
								zap.Stringer("event_type", e.Res.EventType()),
							)
						}
						if e.Message.Offset > latest.Offset {
							latest = e.Message
						}
					case <-ctx.Done():
						return ctx.Err()
					case <-softTicker.C:
						// Finish batch.
						if t.Rows() > 0 {
							a.secSaved.Add(ctx, int64(t.Rows()))
							return nil
						}
					case <-hardTicker.C:
						a.secSaved.Add(ctx, int64(t.Rows()))
						return io.EOF
					}
				}
			},
		}); err != nil {
			_ = db.Close()
			return errors.Wrap(err, "query")
		}
		if err := db.Close(); err != nil {
			return errors.Wrap(err, "close")
		}
		if latest.Offset > 0 {
			// Committing only after query is finished.
			if err := a.reader.Load().CommitMessages(ctx, latest); err != nil {
				return errors.Wrap(err, "commit kafka offset")
			}
			a.secOffsetCommitted.Observe(latest.Offset, metric.WithAttributes(kafkaAttributes(latest, a.reader.Load().Stats())...))
		}
	}
}

func kafkaAttributes(m kafka.Message, stat kafka.ReaderStats) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("kafka.client_id", stat.ClientID),
		attribute.String("kafka.topic", m.Topic),
		attribute.Int("kafka.partition", m.Partition),
	}
}
