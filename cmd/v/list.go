package main

import (
	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
)

func newListCmd(a *Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			apps, err := a.client.GetApplications(ctx)
			if err != nil {
				return errors.Wrap(err, "ListThings")
			}
			for _, app := range apps {
				cmd.Printf("name=%s\tns=%s\n", app.Name, app.Namespace)
			}
			return nil
		},
	}
	return cmd
}
