package main

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/ch-go"
	chProto "github.com/ClickHouse/ch-go/proto"
	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/go-faster/vega/internal/kfk"
)

type Entry[T any] struct {
	Raw     []byte
	Message kafka.Message
	Res     T
}

type Table interface {
	Reset()
	Rows() int
	Insert() string
	Result() chProto.Results
	ResultColumns() []string
	Input() chProto.Input
}

type Metrics struct {
	ParseCount  metric.Int64Counter `name:"parse.count"`
	ParseErrors metric.Int64Counter `name:"parse.errors_count"`

	EntriesRead  metric.Int64Counter `name:"entries.read"`
	EntriesSaved metric.Int64Counter `name:"entries.saved"`

	OffsetRead     metric.Int64Observer `autometric:"-"`
	OffsetCommited metric.Int64Observer `autometric:"-"`
}

type IngesterOptions[M proto.Message, T Table] struct {
	Log       *zap.Logger
	Telemetry *app.Telemetry
	Topic     string
	Group     string
	Servers   []Server
	TableName string
	DDL       string
	Metrics   Metrics

	NewTable    func(tableName string) T
	AppendEntry func(t T, e *Entry[M]) error
	NewMessage  func() M
}

func NewIngester[M proto.Message, T Table](opt IngesterOptions[M, T]) *Ingester[M, T] {
	return &Ingester[M, T]{
		log:       opt.Log,
		telemetry: opt.Telemetry,
		entries:   make(chan *Entry[M], 1000),
		topic:     opt.Topic,
		group:     opt.Group,

		initializeDB: true,
		ddl:          opt.DDL,
		servers:      opt.Servers,
		tableName:    opt.TableName,
		newTable:     opt.NewTable,
		appendEntry:  opt.AppendEntry,
		newMessage:   opt.NewMessage,

		metrics: opt.Metrics,
	}
}

type Ingester[M proto.Message, T Table] struct {
	log       *zap.Logger
	telemetry *app.Telemetry

	entries chan *Entry[M]
	reader  atomic.Pointer[kafka.Reader]
	topic   string
	group   string

	initializeDB bool
	ddl          string
	servers      []Server
	tableName    string
	newTable     func(tableName string) T
	appendEntry  func(t T, e *Entry[M]) error
	newMessage   func() M

	metrics Metrics
}

func (a *Ingester[M, T]) setupClickHouse(ctx context.Context, s Server) error {
	a.log.Info("Setting up ClickHouse",
		zap.String("addr", s.Addr),
		zap.String("db", s.DB),
		zap.String("user", s.User),
	)
	db, err := ch.Dial(ctx, ch.Options{
		Address:  s.Addr,
		User:     s.User,
		Password: s.Password,
		Database: s.DB,
		Logger:   a.log.Named("ch"),

		OpenTelemetryInstrumentation: true,

		MeterProvider:  a.telemetry.MeterProvider(),
		TracerProvider: a.telemetry.TracerProvider(),
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
	ddl := a.ddl
	ddl += "\nTTL toDateTime(timestamp) + INTERVAL 6 HOUR"
	if err := db.Do(ctx, ch.Query{Body: ddl}); err != nil {
		return errors.Wrap(err, "ddl")
	}

	return nil
}

func (a *Ingester[M, T]) Setup(ctx context.Context) error {
	for _, server := range a.servers {
		if err := a.setupClickHouse(ctx, server); err != nil {
			return errors.Wrapf(err, "setup clickhouse %s %s", a.topic, server.Addr)
		}
	}
	return nil
}
func (a *Ingester[M, T]) entry(msg kafka.Message) (*Entry[M], error) {
	f := a.newMessage()

	if err := proto.Unmarshal(msg.Value, f); err != nil {
		return nil, errors.Wrap(err, "unmarshal entries")
	}

	e := &Entry[M]{
		Raw: bytes.Clone(msg.Value),
		Res: f,
	}

	return e, nil
}

type fnLogger func(string, ...interface{})

func (f fnLogger) Printf(s string, i ...interface{}) {
	f(s, i...)
}

func kafkaAttributes(m kafka.Message, stat kafka.ReaderStats) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("kafka.client_id", stat.ClientID),
		attribute.String("kafka.topic", m.Topic),
		attribute.Int("kafka.partition", m.Partition),
	}
}

func (a *Ingester[M, T]) Consume(ctx context.Context) error {
	const (
		kafkaMinSizeBytes = 1014 * 25        // 25kb
		kafkaMaxSizeBytes = 1024 * 1024 * 10 // 10 mb
		kafkaMaxWait      = 3 * time.Second
	)

	lg := a.log.Named(a.topic)
	readerConfig := kafka.ReaderConfig{
		Topic:    a.topic,
		Brokers:  kfk.Addrs(),
		Dialer:   kfk.Dialer(),
		GroupID:  a.group,
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
	a.log.Info("Reader config",
		zap.Strings("brokers", readerConfig.Brokers),
		zap.String("topic", readerConfig.Topic),
		zap.String("group_id", readerConfig.GroupID),
		zap.Int("min_bytes", readerConfig.MinBytes),
		zap.Int("max_bytes", readerConfig.MaxBytes),
		zap.Duration("max_wait", readerConfig.MaxWait),
	)
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
			return errors.Wrap(err, "entry parse")
		}
		a.metrics.EntriesRead.Add(ctx, 1)
		a.metrics.OffsetRead.Observe(msg.Offset, metric.WithAttributes(kafkaAttributes(msg, a.reader.Load().Stats())...))
		select {
		case a.entries <- e:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func clickHouseServer(list []Server) Server {
	return list[rand.Intn(len(list))] // #nosec G404
}

func (a *Ingester[M, T]) Ingest(ctx context.Context) error {
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

	hardTicker := time.NewTicker(ingestHardTimeout)
	defer hardTicker.Stop()
	softTicker := time.NewTicker(ingestSoftTimeout)
	defer softTicker.Stop()

	for {
		s := clickHouseServer(a.servers)
		t := a.newTable(a.tableName)

		db, err := ch.Dial(ctx, ch.Options{
			Logger:      a.log.Named("entries"),
			Address:     s.Addr,
			User:        s.User,
			Password:    s.Password,
			Database:    s.DB,
			Compression: ch.CompressionLZ4,

			OpenTelemetryInstrumentation: true,

			MeterProvider:  a.telemetry.MeterProvider(),
			TracerProvider: a.telemetry.TracerProvider(),
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
						a.metrics.EntriesSaved.Add(ctx, int64(t.Rows()))
						return nil
					}
					select {
					case e := <-a.entries:
						if err := a.appendEntry(t, e); err != nil {
							a.log.Warn("Append entry",
								zap.Error(err),
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
							a.metrics.EntriesSaved.Add(ctx, int64(t.Rows()))
							return nil
						}
					case <-hardTicker.C:
						a.metrics.EntriesSaved.Add(ctx, int64(t.Rows()))
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
			a.metrics.OffsetCommited.Observe(latest.Offset, metric.WithAttributes(kafkaAttributes(latest, a.reader.Load().Stats())...))
		}
	}
}
