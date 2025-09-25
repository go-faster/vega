package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-faster/errors"

	"github.com/go-faster/vega/internal/installer"
)

func file(name string) string {
	return filepath.Join("_hack", name)
}

func run(ctx context.Context) error {
	// Actually this can be implemented as an acyclic graph.
	kubeConfig := filepath.Join("_out", "kubeconfig.yml")
	steps := []installer.Step{
		&installer.Parallel{
			Max: 6,
			Steps: []installer.Step{
				installer.BuildBinary("vega-agent"),
				installer.BuildBinary("vega-ingest"),
				installer.BuildBinary("v"),
				installer.BuildBinary("vega"),
				installer.BuildBinary("vega-create-buckets"),
			},
		},
		&installer.Parallel{
			Max: 6,
			Steps: []installer.Step{
				&installer.Docker{
					Tags:    []string{"vega"},
					File:    "Dockerfile",
					Context: ".",
				},
				&installer.Docker{
					Tags:    []string{"vega-agent"},
					File:    "agent.Dockerfile",
					Context: ".",
				},
				&installer.Docker{
					Tags:    []string{"vega-ingest"},
					File:    "ingest.Dockerfile",
					Context: ".",
				},
				&installer.Docker{
					Tags:    []string{"vega-create-buckets"},
					File:    "create-buckets.Dockerfile",
					Context: ".",
				},
			},
		},
		&installer.Kind{
			Name:       "vega",
			Config:     file("vega.kind.yml"),
			KubeConfig: kubeConfig,
		},
		&installer.KindLoad{
			Name:       "vega",
			Images:     []string{"vega", "vega-agent", "vega-ingest", "vega-create-buckets"},
			KubeConfig: kubeConfig,
			Nodes:      []string{"vega-worker"},
		},
		&installer.DockerPull{
			ImagesFile: file("images.txt"),
		},
		&installer.KubeApply{
			File:       file("monitoring.coreos.com_servicemonitors.yaml"),
			KubeConfig: kubeConfig,
		},
		&installer.KubeApply{
			File:       file("monitoring.coreos.com_podmonitors.yaml"),
			KubeConfig: kubeConfig,
		},
		&installer.KindLoad{
			Name:       "vega",
			ImagesFile: file("images.txt"),
			KubeConfig: kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "cilium",
			Chart:           "cilium/cilium",
			Install:         true,
			Version:         "1.18.2",
			Values:          file("cilium.yml"),
			Namespace:       "cilium",
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.CiliumStatus{
			Namespace:  "cilium",
			Wait:       true,
			KubeConfig: kubeConfig,
		},
		&installer.Parallel{
			Max: 6,
			Steps: []installer.Step{
				&installer.HelmUpgrade{
					Name:            "tetragon",
					Chart:           "cilium/tetragon",
					Install:         true,
					Version:         "1.5.0",
					Values:          file("tetragon.yml"),
					Namespace:       "cilium",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "ch",
					Chart:           "clickhouse-operator/altinity-clickhouse-operator",
					Install:         true,
					Namespace:       "clickhouse",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "0.25.3",
				},
				&installer.HelmUpgrade{
					Name:            "otel",
					Chart:           "faster/oteldb",
					Install:         true,
					Version:         "0.19.1",
					Values:          file("oteldb.yml"),
					Namespace:       "faster",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "grafana",
					Chart:           "grafana/grafana",
					Install:         true,
					Version:         "9.4.5",
					Values:          file("grafana.yml"),
					Namespace:       "monitoring",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "vmo",
					Chart:           "vm/victoria-metrics-operator",
					Install:         true,
					Values:          file("vm.operator.yml"),
					Namespace:       "vm",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "0.53.0",
				},
				&installer.HelmUpgrade{
					Name:            "nats",
					Chart:           "nats/nats",
					Install:         true,
					Namespace:       "nats",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "1.2.10",
				},
				&installer.HelmUpgrade{
					Name:            "operator",
					Chart:           "minio-operator/operator",
					Install:         true,
					Namespace:       "minio",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "7.0.0",
				},
				&installer.HelmUpgrade{
					Name:            "loki",
					Chart:           "grafana/loki",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("loki.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "6.40.0",
				},
				&installer.HelmUpgrade{
					Name:            "tempo",
					Chart:           "grafana/tempo",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("tempo.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "1.23.3",
				},
				&installer.HelmUpgrade{
					Name:            "pyroscope",
					Chart:           "grafana/pyroscope",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("pyroscope.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "1.12.0",
				},
				&installer.HelmUpgrade{
					Name:            "kube-state",
					Chart:           "prometheus-community/kube-state-metrics",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("kube-state.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "5.29.0",
				},
				&installer.HelmUpgrade{
					Name:            "ingress-nginx",
					Chart:           "ingress-nginx",
					Install:         true,
					Namespace:       "ingress-nginx",
					Values:          file("nginx.yml"),
					CreateNamespace: true,
					Repo:            "https://kubernetes.github.io/ingress-nginx",
					KubeConfig:      kubeConfig,
					Version:         "4.12.0",
				},
				&installer.HelmUpgrade{
					Name:            "trivy-operator",
					Chart:           "aqua/trivy-operator",
					Install:         true,
					Namespace:       "trivy-system",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
					Version:         "0.30.0",
				},
				&installer.HelmUpgrade{
					Name:            "harbor",
					Chart:           "harbor/harbor",
					Install:         true,
					Namespace:       "harbor",
					Values:          file("harbor.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "agent",
					Chart:           "open-telemetry/opentelemetry-collector",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("otel-collector.agent.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "aggregator",
					Chart:           "open-telemetry/opentelemetry-collector",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("otel-collector.aggregator.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "operator",
					Chart:           "open-telemetry/opentelemetry-operator",
					Install:         true,
					Namespace:       "monitoring",
					Values:          file("otel-operator.yml"),
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
			},
		},
		&installer.KubeRolloutStatus{
			Target:     "deployment",
			Name:       "ingress-nginx-controller",
			Namespace:  "ingress-nginx",
			Watch:      true,
			KubeConfig: kubeConfig,
		},
		&installer.KubeApply{
			File:       file("k8s"),
			KubeConfig: kubeConfig,
		},
		&installer.KubeCreate{
			File:       file("k8s-create"),
			KubeConfig: kubeConfig,
		},
	}
	for _, step := range steps {
		fmt.Println("step:", step.Step().Name)
		if err := step.Run(ctx); err != nil {
			return errors.Wrap(err, step.Step().Name)
		}
	}
	fmt.Println("DONE")
	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(2)
	}
}
