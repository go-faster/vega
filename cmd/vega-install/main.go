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
				installer.BuildBinary("create-buckets"),
			},
		},
		&installer.Parallel{
			Max: 6,
			Steps: []installer.Step{
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
					Tags:    []string{"create-buckets"},
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
			Images:     []string{"vega-agent", "vega-ingest", "create-buckets"},
			KubeConfig: kubeConfig,
		},
		&installer.DockerPull{
			ImagesFile: file("images.txt"),
		},
		&installer.KubeApply{
			File:       file("monitoring.coreos.com_servicemonitors.yaml"),
			KubeConfig: kubeConfig,
		},
		&installer.Parallel{
			Max: 6,
			Steps: []installer.Step{
				&installer.KindLoad{
					Name:       "vega",
					ImagesFile: file("images.txt"),
					KubeConfig: kubeConfig,
				},
				&installer.HelmUpgrade{
					Name:            "cilium",
					Chart:           "cilium/cilium",
					Install:         true,
					Version:         "1.16.5",
					Values:          file("cilium.yml"),
					Namespace:       "cilium",
					CreateNamespace: true,
					KubeConfig:      kubeConfig,
				},
			},
		},
		&installer.CiliumStatus{
			Namespace:  "cilium",
			Wait:       true,
			KubeConfig: kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "tetragon",
			Chart:           "cilium/tetragon",
			Install:         true,
			Version:         "1.3.0",
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
			Version:         "0.24.0",
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
			Version:         "8.8.2",
			Values:          file("grafana.yml"),
			Namespace:       "monitoring",
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "monitoring",
			Chart:           "grafana/k8s-monitoring",
			Install:         true,
			Values:          file("k8s-monitoring.yml"),
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
		},
		&installer.HelmUpgrade{
			Name:            "nats",
			Chart:           "nats/nats",
			Install:         true,
			Namespace:       "nats",
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "operator",
			Chart:           "minio-operator/operator",
			Install:         true,
			Namespace:       "minio",
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "operator",
			Chart:           "minio-operator/operator",
			Install:         true,
			Namespace:       "minio",
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "loki",
			Chart:           "grafana/loki",
			Install:         true,
			Namespace:       "monitoring",
			Values:          file("loki.yml"),
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "tempo",
			Chart:           "grafana/tempo",
			Install:         true,
			Namespace:       "monitoring",
			Values:          file("tempo.yml"),
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "oncall",
			Chart:           "grafana/oncall",
			Install:         true,
			Namespace:       "monitoring",
			Values:          file("oncall.yml"),
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.HelmUpgrade{
			Name:            "kube-state",
			Chart:           "prometheus-community/kube-state-metrics",
			Install:         true,
			Namespace:       "monitoring",
			Values:          file("kube-state.yml"),
			CreateNamespace: true,
			KubeConfig:      kubeConfig,
		},
		&installer.KubeApply{
			File:       file("k8s"),
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
