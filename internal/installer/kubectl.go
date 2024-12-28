package installer

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

type KubeApply struct {
	Bin  string
	File string
}

func (k KubeApply) Step() StepInfo {
	return StepInfo{Name: "kubectl apply -f " + k.File}
}

func (k KubeApply) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = "kubectl"
	}
	arg := []string{
		"apply", "-f", k.File,
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl apply -f")
	}
	return nil
}
