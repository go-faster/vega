package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func root() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "v",
		Short:         "cli for vega",
		Long:          "TUI and CLI for vega platform",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Hello")
			return nil
		},
	}
	return cmd
}

func main() {
	cmd := root()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
