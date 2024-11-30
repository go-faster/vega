package installer

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

// Kind is Kubernetes In Docker (KIND) installer.
type Kind struct {
	Bin string
}

func (k Kind) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = "kind"
	}
	arg := []string{
		"create", "cluster",
		"-n", "vega",
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "create cluster")
	}
	return nil
}

func (k Kind) Step() StepInfo {
	return StepInfo{Name: "kind"}
}
