package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-faster/errors"

	"github.com/go-faster/vega/internal/installer"
)

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
		&installer.Kind{},
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
