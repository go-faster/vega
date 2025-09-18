package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

// Run runs f.
func Run(f func(ctx context.Context) error) {
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := f(ctx)
	duration := time.Since(start).Round(time.Millisecond)
	if duration > time.Second*5 {
		duration = duration.Round(time.Second)
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr,
			color.New(color.Bold, color.FgRed).Sprint("error"),
			color.New(color.FgRed, color.Faint).Sprintf("(%s):", duration),
			color.New(color.FgRed).Sprintf("%+v", err),
		)
		os.Exit(2)
	}

	fmt.Println(
		color.New(color.Bold, color.FgGreen).Sprint("OK"),
		color.New(color.FgGreen).Sprint(duration),
	)
}
