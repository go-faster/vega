package main

import (
	"fmt"
	"os"

	"github.com/go-faster/errors"
	"github.com/spf13/cobra"

	"github.com/go-faster/vega/internal/oas"
)

// Application wraps state and initialized dependencies for vega client.
type Application struct {
	client *oas.Client
}

func root() *cobra.Command {
	app := &Application{}
	cmd := &cobra.Command{
		Use:           "v",
		Short:         "cli for vega",
		Long:          "TUI and CLI for vega platform",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			client, err := oas.NewClient("http://vega.localhost")
			if err != nil {
				return errors.Wrap(err, "oas.NewClient")
			}
			app.client = client
			return nil
		},
	}
	cmd.AddCommand(newVersionCmd(app))
	cmd.AddCommand(newWaitCmd(app))
	cmd.AddCommand(newListCmd(app))
	cmd.AddCommand(newGetCmd(app))
	return cmd
}

func main() {
	cmd := root()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
