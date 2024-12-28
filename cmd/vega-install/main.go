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
			Name:   "vega",
			Config: file("vega.kind.yml"),
		},
		&installer.KubeApply{
			File: file("monitoring.coreos.com_servicemonitors.yaml"),
		},
		&installer.HelmUpgrade{
			Name:            "cilium",
			Chart:           "cilium/cilium",
			Install:         true,
			Version:         "1.16.5",
			Values:          file("cilium.yml"),
			Namespace:       "cilium",
			CreateNamespace: true,
		},
		&installer.CiliumStatus{
			Namespace: "cilium",
			Wait:      true,
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
