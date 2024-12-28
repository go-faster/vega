package installer

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

type CiliumStatus struct {
	Bin       string
	Namespace string
	Wait      bool
}

func (c CiliumStatus) Step() StepInfo {
	return StepInfo{Name: "cilium status"}
}

func (c CiliumStatus) Run(ctx context.Context) error {
	b := c.Bin
	if b == "" {
		b = "cilium"
	}
	arg := []string{
		"status",
	}
	if c.Wait {
		arg = append(arg, "--wait")
	}
	if c.Namespace != "" {
		arg = append(arg, "--namespace", c.Namespace)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "cilium status")
	}
	return nil
}
