package installer

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type funcStep func(ctx context.Context) error

func (f funcStep) Run(ctx context.Context) error { return f(ctx) }

func (f funcStep) Step() StepInfo {
	return StepInfo{Name: "funcStep"}
}

func TestParallel(t *testing.T) {
	p := Parallel{
		Max: 5,
	}
	var (
		current int
		total   int
		mux     sync.Mutex
	)
	f := func(ctx context.Context) error {
		mux.Lock()
		current++
		total++
		assert.LessOrEqual(t, current, p.Max)
		mux.Unlock()
		defer func() {
			mux.Lock()
			current--
			mux.Unlock()
		}()
		return nil
	}
	for i := 0; i < 100; i++ {
		p.Steps = append(p.Steps, funcStep(f))
	}
	assert.NoError(t, p.Run(context.Background()))
	assert.Equal(t, 0, current)
	assert.Equal(t, 100, total)
	assert.NotEmpty(t, p.Step().Name)
}
