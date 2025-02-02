// Package api implements vega API handler.
package api

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/kubernetes"

	"github.com/go-faster/vega/internal/oas"
)

var _ oas.Handler = (*Handler)(nil)

type Handler struct {
	kube  *kubernetes.Clientset
	trace trace.Tracer
}

func (h Handler) GetHealth(ctx context.Context) (*oas.Health, error) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, errors.New("failed to read build info")
	}

	var commit string
	var buildDate time.Time

	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			commit = setting.Value
		case "vcs.time":
			buildDate, _ = time.Parse(time.RFC3339, setting.Value)
		case "vcs.modified":
			if setting.Value == "true" {
				commit += "-modified"
			}
		}
	}

	return &oas.Health{
		Status:    "ok",
		Version:   buildInfo.Main.Version,
		BuildDate: buildDate,
		Commit:    commit,
	}, nil
}
func (h Handler) NewError(ctx context.Context, err error) *oas.ErrorStatusCode {
	var (
		traceID oas.OptTraceID
		spanID  oas.OptSpanID
	)
	if span := trace.SpanFromContext(ctx).SpanContext(); span.HasTraceID() {
		traceID = oas.NewOptTraceID(oas.TraceID(span.TraceID().String()))
		spanID = oas.NewOptSpanID(oas.SpanID(span.SpanID().String()))
	}
	return &oas.ErrorStatusCode{
		StatusCode: 500,
		Response: oas.Error{
			ErrorMessage: err.Error(),
			TraceID:      traceID,
			SpanID:       spanID,
		},
	}
}

func NewHandler(
	kube *kubernetes.Clientset,
	traceProvider trace.TracerProvider,
) *Handler {
	return &Handler{
		kube:  kube,
		trace: traceProvider.Tracer("vega.api"),
	}
}
