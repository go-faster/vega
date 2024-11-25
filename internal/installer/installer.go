// Package installer implements development installer for vega.
package installer

import "context"

// StepInfo wraps step information description.
type StepInfo struct {
	Name string
}

// Step of setup.
type Step interface {
	Run(ctx context.Context) error
	Step() StepInfo
}
