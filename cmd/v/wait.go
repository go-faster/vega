package main

import (
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
)

func newWaitCmd(a *Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for the vega api to be ready",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			bo := backoff.NewExponentialBackOff()
			bo.MaxInterval = time.Second
			bo.MaxElapsedTime = time.Minute * 15
			bo.InitialInterval = time.Millisecond * 100

			if err := backoff.RetryNotify(func() error {
				_, err := a.client.GetHealth(ctx)
				return err
			}, bo, func(err error, duration time.Duration) {
				cmd.Printf("Waiting for vega api to be ready: %v\n", err)
			}); err != nil {
				return errors.Wrap(err, "GetHealth")
			}

			cmd.Println("Vega api is ready")
			return nil
		},
	}
	return cmd
}
