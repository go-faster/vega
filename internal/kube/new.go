package kube

import (
	"net/http"
	"os"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/go-faster/sdk/app"
)

type rtf func(*http.Request) (*http.Response, error)

func (p rtf) RoundTrip(req *http.Request) (*http.Response, error) {
	return p(req)
}

func NewConfig(t *app.Telemetry) (config *rest.Config, err error) {
	if path, ok := os.LookupEnv("KUBECONFIG_PATH"); ok {
		config, err = clientcmd.BuildConfigFromFlags("", path)
		if err != nil {
			return nil, errors.Wrap(err, "build config")
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "cluster config")
		}
	}

	// Configure trace propagation to kubeapi.
	config = OtelForConfig(config, t)

	return config, nil
}

// OtelForConfig configures otel tracing for the given k8s rest config.
func OtelForConfig(config *rest.Config, t *app.Telemetry) *rest.Config {
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return rtf(func(request *http.Request) (*http.Response, error) {
			return rt.RoundTrip(request)
		})
	})
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return otelhttp.NewTransport(rt,
			otelhttp.WithTracerProvider(t.TracerProvider()),
			otelhttp.WithMeterProvider(t.MeterProvider()),
			otelhttp.WithPropagators(t.TextMapPropagator()),
		)
	})
	return config
}

func New(t *app.Telemetry) (*kubernetes.Clientset, error) {
	config, err := NewConfig(t)
	if err != nil {
		return nil, errors.Wrap(err, "new config")
	}
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "k8s client")
	}

	return kubeClient, nil
}
