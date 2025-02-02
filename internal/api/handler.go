// Package api implements vega API handler.
package api

import (
	"context"
	"fmt"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/zctx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/go-faster/vega/internal/oas"
	"github.com/go-faster/vega/internal/promapi"
	"github.com/go-faster/vega/internal/semconv"
)

var _ oas.Handler = (*Handler)(nil)

type Handler struct {
	kube  *kubernetes.Clientset
	prom  *promapi.Client
	trace trace.Tracer
}

func toPrometheusTimestamp(t time.Time) promapi.PrometheusTimestamp {
	return promapi.PrometheusTimestamp(strconv.FormatInt(t.Unix(), 10))
}

func toOptPrometheusTimestamp(t time.Time) promapi.OptPrometheusTimestamp {
	if t.IsZero() {
		return promapi.OptPrometheusTimestamp{}
	}
	return promapi.NewOptPrometheusTimestamp(toPrometheusTimestamp(t))
}

func (h *Handler) getInstantQuery(ctx context.Context, now time.Time, query string) (float64, error) {
	ctx, span := h.trace.Start(ctx, "getInstantQuery",
		trace.WithAttributes(
			attribute.String("query", query),
		),
	)
	defer span.End()

	result, err := h.prom.GetQuery(ctx, promapi.GetQueryParams{
		Query: query,
		Time:  toOptPrometheusTimestamp(now),
	})
	if err != nil {
		return 0, errors.Wrap(err, "get query")
	}
	zctx.From(ctx).Info("Get query",
		zap.String("query", query),
		zap.Any("result", result),
	)
	var out float64
	for _, res := range result.Data.Vector.Result {
		out += res.Value.HistogramOrValue.StringFloat64
	}
	return out, nil
}

func (h *Handler) getPodResources(ctx context.Context, pod v1.Pod) (*oas.PodResources, error) {
	ctx, span := h.trace.Start(ctx, "getPodResources",
		trace.WithAttributes(
			attribute.String("namespace", pod.Namespace),
			attribute.String("pod", pod.Name),
		),
	)
	defer span.End()
	now := time.Now()
	var out oas.PodResources
	{
		query := fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace=%q, pod=~%q, image!="", container!=""}) by (container, id)`,
			pod.Namespace, pod.Name,
		)
		memUsageTotal, err := h.getInstantQuery(ctx, now, query)
		if err != nil {
			return nil, errors.Wrap(err, "get memory usage")
		}
		out.MemUsageTotalBytes = int64(memUsageTotal)
	}
	{
		query := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace=%q, pod=~%q, image!="", container!="", cluster=""}[30s])) by (container, id)`,
			pod.Namespace, pod.Name,
		)
		cpuUsageTotal, err := h.getInstantQuery(ctx, now, query)
		if err != nil {
			return nil, errors.Wrap(err, "get cpu usage")
		}
		out.CPUUsageTotalMillicores = cpuUsageTotal
	}
	return &out, nil
}

func (h *Handler) GetApplication(ctx context.Context, params oas.GetApplicationParams) (*oas.ApplicationSummary, error) {
	appList, err := h.getApplications(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get applications")
	}

	var app oas.Application
	for _, a := range appList {
		if a.Name == params.Name {
			app = a
			break
		}
	}
	if app.Name == "" {
		return nil, &oas.ErrorStatusCode{
			StatusCode: 404,
			Response: oas.Error{
				ErrorMessage: "application not found",
			},
		}
	}

	summary := &oas.ApplicationSummary{
		Name:      app.Name,
		Namespace: app.Namespace,
	}
	{
		// Fetch pods.
		pods, err := h.kube.CoreV1().Pods(app.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: semconv.LabelVegaApp + "=" + app.Name,
		})
		if err != nil {
			return nil, errors.Wrap(err, "list pods")
		}
		for _, pod := range pods.Items {
			res, err := h.getPodResources(ctx, pod)
			if err != nil {
				return nil, errors.Wrap(err, "get pod resources")
			}
			summary.Pods = append(summary.Pods, oas.Pod{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    string(pod.Status.Phase),
				Resources: *res,
			})
		}
	}

	return summary, nil
}

func (h *Handler) getApplications(ctx context.Context) ([]oas.Application, error) {
	ctx, span := h.trace.Start(ctx, "getApplications")
	defer span.End()

	listOptions := metav1.ListOptions{
		LabelSelector: semconv.LabelVegaApp,
	}
	namespaces, err := h.kube.CoreV1().Namespaces().List(ctx, listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "listing namespaces")
	}

	var mux sync.Mutex
	appMap := make(map[string]oas.Application)
	g, ctx := errgroup.WithContext(ctx)
	for _, ns := range namespaces.Items {
		g.Go(func() error {
			pods, err := h.kube.CoreV1().Pods(ns.Name).List(ctx, listOptions)
			if err != nil {
				return errors.Wrapf(err, "listing pods in namespace %s", ns.Name)
			}
			for _, pod := range pods.Items {
				app := oas.Application{
					Name:      pod.Labels[semconv.LabelVegaApp],
					Namespace: pod.Namespace,
				}
				mux.Lock()
				appMap[app.Name] = app
				mux.Unlock()
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, errors.Wrap(err, "getting applications")
	}
	var appList oas.ApplicationList
	for _, app := range appMap {
		appList = append(appList, app)
	}
	slices.SortFunc(appList, func(a, b oas.Application) int {
		return strings.Compare(a.Name, b.Name)
	})

	return appList, nil
}

func (h *Handler) GetApplications(ctx context.Context) (oas.ApplicationList, error) {
	return h.getApplications(ctx)
}

func (h *Handler) GetHealth(ctx context.Context) (*oas.Health, error) {
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
func (h *Handler) NewError(ctx context.Context, err error) *oas.ErrorStatusCode {
	var (
		traceID oas.OptTraceID
		spanID  oas.OptSpanID
	)
	if span := trace.SpanFromContext(ctx).SpanContext(); span.HasTraceID() {
		traceID = oas.NewOptTraceID(oas.TraceID(span.TraceID().String()))
		spanID = oas.NewOptSpanID(oas.SpanID(span.SpanID().String()))
	}
	if v, ok := errors.Into[*oas.ErrorStatusCode](err); ok {
		v.Response.TraceID = traceID
		v.Response.SpanID = spanID
		if v.StatusCode == 0 {
			v.StatusCode = 500
		}
		if v.Response.ErrorMessage == "" {
			v.Response.ErrorMessage = "internal error"
		}
		return v
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
	promClient *promapi.Client,
	traceProvider trace.TracerProvider,
) *Handler {
	return &Handler{
		kube:  kube,
		prom:  promClient,
		trace: traceProvider.Tracer("vega.api"),
	}
}
