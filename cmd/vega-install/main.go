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
				installer.BuildBinary("v"),
				installer.BuildBinary("vega"),
			},
		},
		&installer.Docker{
			Tags:    []string{"vega-agent:latest"},
			File:    "agent.Dockerfile",
			Context: ".",
		},
		&installer.Kind{
			Name:       "vega",
			Config:     file("vega.kind.yml"),
			KubeConfig: kubeConfig,
		},
		&installer.KindLoad{
			Name:       "vega",
			Images:     []string{"vega-agent:latest"},
			KubeConfig: kubeConfig,
		},
		&installer.KubeApply{
			File:       file("monitoring.coreos.com_servicemonitors.yaml"),
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
		&installer.CiliumStatus{
			Namespace:  "cilium",
			Wait:       true,
			KubeConfig: kubeConfig,
		},
		&installer.DaemonSet{
			Name:       "vega-agent",
			Image:      "vega-agent:latest",
			Namespace:  "default",
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
