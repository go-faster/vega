package main

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-faster/errors"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// KafkaAddrs returns kafka addresses from environment variable.
func KafkaAddrs() []string {
	v := os.Getenv("KAFKA_ADDR")
	var list []string
	for _, s := range strings.Split(v, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			list = append(list, s)
		}
	}
	return list
}

func KafkaDialer() *kafka.Dialer {
	d := &kafka.Dialer{
		KeepAlive: time.Second * 10,
		Timeout:   time.Second * 3,
	}
	if user := os.Getenv("KAFKA_USER"); user != "" {
		d.SASLMechanism = plain.Mechanism{
			Username: user,
			Password: os.Getenv("KAFKA_PASSWORD"),
		}
	}
	return d
}

// KafkaTransport from environment.
func KafkaTransport() *kafka.Transport {
	d := KafkaDialer()
	return &kafka.Transport{
		Dial:        d.DialFunc,
		DialTimeout: d.Timeout,
		SASL:        d.SASLMechanism,
	}
}

// KafkaBalancer from environment.
func KafkaBalancer() kafka.Balancer {
	switch os.Getenv("KAFKA_BALANCER") {
	case "", "least_bytes":
		return &kafka.LeastBytes{}
	case "hash":
		return &kafka.Hash{}
	case "round_robin":
		return &kafka.RoundRobin{}
	default:
		panic("unknown balancer: " + os.Getenv("KAFKA_BALANCER"))
	}
}

type KafkaProducer struct {
	addrs          []string
	lg             *zap.Logger
	writers        map[string]*kafka.Writer
	mux            sync.Mutex
	messagesFailed metric.Int64Counter
	messagesSent   metric.Int64Counter
}

func (k *KafkaProducer) writer(topic string) (*kafka.Writer, error) {
	k.mux.Lock()
	defer k.mux.Unlock()
	writer, ok := k.writers[topic]
	if !ok {
		var err error
		writer, err = k.newWriter(topic)
		if err != nil {
			return nil, errors.Wrap(err, "create kafka writer")
		}
		k.writers[topic] = writer
	}
	return writer, nil
}

func (k *KafkaProducer) newWriter(topic string) (*kafka.Writer, error) {
	lg := k.lg.Named(topic)
	return &kafka.Writer{
		Async:        true,
		BatchSize:    10_000,
		BatchTimeout: time.Second,

		Addr:      kafka.TCP(k.addrs...),
		Topic:     topic,
		Balancer:  KafkaBalancer(),
		Transport: KafkaTransport(),

		Logger: fnLogger(func(s string, i ...interface{}) {
			lg.Sugar().Debugf(s, i...)
		}),
		ErrorLogger: fnLogger(func(s string, i ...interface{}) {
			lg.Sugar().Errorf(s, i...)
		}),
		Completion: func(messages []kafka.Message, err error) {
			ctx := context.Background()
			count := int64(len(messages))
			withAttributes := metric.WithAttributes(
				attribute.String("topic", topic),
			)
			if err == nil {
				k.messagesSent.Add(ctx, count, withAttributes)
				lg.Debug("Kafka messages completed",
					zap.Error(err), zap.Int64("messages.count", count),
				)
			} else {
				k.messagesFailed.Add(ctx, count, withAttributes)
				lg.Error("Kafka messages failed",
					zap.Error(err), zap.Int64("messages.count", count),
				)
			}
		},
	}, nil
}

type fnLogger func(string, ...interface{})

func (f fnLogger) Printf(s string, i ...interface{}) {
	f(s, i...)
}

func NewKafkaProducer(lg *zap.Logger, provider metric.MeterProvider) (*KafkaProducer, error) {
	addrs := KafkaAddrs()
	lg.Info("Initializing kafka producer",
		zap.Strings("addrs", addrs),
		zap.Int("addrs.count", len(addrs)),
	)
	meter := provider.Meter("kafka.producer")
	messagesSent, err := meter.Int64Counter("kafka.messages.sent")
	if err != nil {
		return nil, errors.Wrap(err, "register metric")
	}
	messagesFailed, err := meter.Int64Counter("kafka.messages.failed")
	if err != nil {
		return nil, errors.Wrap(err, "register metric")
	}
	k := &KafkaProducer{
		addrs:   addrs,
		lg:      lg,
		writers: make(map[string]*kafka.Writer),

		messagesSent:   messagesSent,
		messagesFailed: messagesFailed,
	}
	return k, nil
}

func (k *KafkaProducer) Produce(ctx context.Context, topic string, msg proto.Message) error {
	writer, err := k.writer(topic)
	if err != nil {
		return errors.Wrap(err, "get kafka writer")
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}
	return writer.WriteMessages(ctx,
		kafka.Message{
			Value: data,
		},
	)
}

func (k *KafkaProducer) Close() error {
	var errs []error
	for _, writer := range k.writers {
		if err := writer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return multierr.Combine(errs...)
}
