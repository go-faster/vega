package installer

import (
	"context"
	"strings"

	"golang.org/x/sync/errgroup"
)

// Parallel runs all steps in parallel.
type Parallel struct {
	Steps []Step
	Max   int
}

// Step returns step information.
func (p *Parallel) Step() StepInfo {
	var b strings.Builder
	for i, s := range p.Steps {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(s.Step().Name)
	}
	return StepInfo{
		Name: "Parallel: " + b.String(),
	}
}

// Run all steps in parallel.
func (p *Parallel) Run(ctx context.Context) error {
	n := p.Max
	if p.Max == 0 {
		n = len(p.Steps)
	}
	g, ctx := errgroup.WithContext(ctx)
	sema := make(chan struct{}, n)
	for _, s := range p.Steps {
		g.Go(func() error {
			sema <- struct{}{}
			defer func() {
				<-sema
			}()
			return s.Run(ctx)
		})
	}
	return g.Wait()
}
