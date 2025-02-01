package main

import (
	"os"
	"sync"

	"github.com/go-faster/errors"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Producer struct {
	addrs          []string
	lg             *zap.Logger
	mux            sync.Mutex
	messagesFailed metric.Int64Counter
	messagesSent   metric.Int64Counter
	nc             *nats.Conn
}

func NewProducer(lg *zap.Logger, provider metric.MeterProvider) (*Producer, error) {
	addr := os.Getenv("NATS_URL")
	nc, err := nats.Connect(addr)
	if err != nil {
		return nil, errors.Wrap(err, "connect")
	}
	lg.Info("Initializing producer")
	meter := provider.Meter("kafka.producer")
	messagesSent, err := meter.Int64Counter("nats.messages.sent")
	if err != nil {
		return nil, errors.Wrap(err, "register metric")
	}
	messagesFailed, err := meter.Int64Counter("nats.messages.failed")
	if err != nil {
		return nil, errors.Wrap(err, "register metric")
	}
	k := &Producer{
		nc: nc,
		lg: lg,

		messagesSent:   messagesSent,
		messagesFailed: messagesFailed,
	}
	return k, nil
}

func (k *Producer) Produce(subject string, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}
	return k.nc.PublishMsg(&nats.Msg{
		Data:    data,
		Subject: subject,
	})
}

func (k *Producer) Close() error {
	k.nc.Close()
	return nil
}
