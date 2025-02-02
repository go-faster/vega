package api

import (
	"context"
	"github.com/go-faster/vega/internal/oas"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/kubernetes"
)

var _ oas.Handler = (*Handler)(nil)

type Handler struct {
	kube  *kubernetes.Clientset
	trace trace.Tracer
}

func (h Handler) GetHealth(ctx context.Context) (*oas.Health, error) {
	return &oas.Health{
		Status: "ok",
	}, nil
}

func (h Handler) NewError(_ context.Context, err error) *oas.ErrorStatusCode {
	return &oas.ErrorStatusCode{
		StatusCode: 500,
		Response: oas.Error{
			ErrorMessage: err.Error(),
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
