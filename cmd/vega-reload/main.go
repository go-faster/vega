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
	kubeConfig := filepath.Join("_out", "kubeconfig.yml")
	steps := []installer.Step{
		&installer.Parallel{
			Max: 6,
			Steps: []installer.Step{
				installer.BuildBinary("vega-agent"),
				installer.BuildBinary("vega-ingest"),
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
		&installer.KindLoad{
			Name:       "vega",
			Images:     []string{"vega", "vega-agent", "vega-ingest", "vega-create-buckets"},
			KubeConfig: kubeConfig,
		},
		&installer.KubeApply{
			File:       file("k8s"),
			KubeConfig: kubeConfig,
		},
		&installer.KubeDelete{
			File:       file("k8s-create"),
			KubeConfig: kubeConfig,
		},
		&installer.KubeCreate{
			File:       file("k8s-create"),
			KubeConfig: kubeConfig,
		},
		&installer.KubeRestart{
			Target:     "daemonset",
			Name:       "vega-agent",
			Namespace:  "vega",
			KubeConfig: kubeConfig,
		},
		&installer.KubeRestart{
			Target:     "deployment",
			Name:       "vega-ingest",
			Namespace:  "vega",
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
