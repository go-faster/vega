// Package kfk wraps kafka helpers.
package kfk

import (
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// Addrs returns kafka addresses from environment variable.
func Addrs() []string {
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

func Dialer() *kafka.Dialer {
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

// Transport from environment.
func Transport() *kafka.Transport {
	d := Dialer()
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
