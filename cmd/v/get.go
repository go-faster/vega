package main

import (
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
			cmd.Printf("%s %s\n", app.Name, app.Namespace)
			cmd.Printf("pods:\n")
			for _, pod := range app.Pods {
				cmd.Printf("  %s\n", pod.Name)
			}
			return nil
		},
	}
	return cmd
}
