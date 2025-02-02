package main

import (
	"github.com/dustin/go-humanize"
	"github.com/go-faster/errors"
	"github.com/spf13/cobra"

	"github.com/go-faster/vega/internal/oas"
)

func newGetCmd(a *Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			app, err := a.client.GetApplication(ctx, oas.GetApplicationParams{
				Name: args[0],
			})
			if err != nil {
				return errors.Wrap(err, "GetApplication")
			}
			cmd.Printf("%s (ns=%s)\n", app.Name, app.Namespace)
			cmd.Printf("pods:\n")
			for _, pod := range app.Pods {
				cmd.Printf("  %s (mem=%s, cpu=%f)\n",
					pod.Name,
					humanize.Bytes(uint64(pod.Resources.MemUsageTotalBytes)),
					pod.Resources.CPUUsageTotalMillicores,
				)
			}
			return nil
		},
	}
	return cmd
}
