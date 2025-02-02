package main

import (
	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
)

func newVersionCmd(a *Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			h, err := a.client.GetHealth(ctx)
			if err != nil {
				return errors.Wrap(err, "GetHealth")
			}
			cmd.Printf("Version: %s\n", h.Version)
			cmd.Printf("Commit: %s\n", h.Commit)
			cmd.Printf("Build Date: %s\n", h.BuildDate)
			return nil
		},
	}
	return cmd
}
