package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/go-faster/errors"
	"github.com/spf13/cobra"
)

func newWaitCmd(a *Application) *cobra.Command {
	var arg struct {
		Duration time.Duration
	}
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for the vega api to be ready",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			bo := backoff.NewExponentialBackOff()
			bo.MaxInterval = time.Second
			bo.MaxElapsedTime = arg.Duration
			bo.InitialInterval = time.Millisecond * 100

			if err := backoff.RetryNotify(func() error {
				_, err := a.client.GetHealth(ctx)
				return err
			}, bo, func(err error, duration time.Duration) {
				cmd.Printf("Waiting for vega api to be ready: %v\n", err)
			}); err != nil {
				res, err := http.Get("http://vega.localhost/health")
				if err != nil {
					return errors.Wrap(err, "http.Get")
				}
				defer func() {
					_ = res.Body.Close()
				}()
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), res.Status)
				_, _ = io.Copy(cmd.OutOrStdout(), res.Body)
				return errors.Wrap(err, "GetHealth")
			}

			cmd.Println("Vega api is ready")
			return nil
		},
	}
	cmd.Flags().DurationVarP(&arg.Duration, "duration", "d", time.Second*15, "Maximum time to wait for the vega api to be ready")
	return cmd
}
