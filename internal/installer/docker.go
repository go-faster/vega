package installer

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

type Docker struct {
	Bin     string
	Tags    []string
	File    string
	Context string
}

func (d Docker) Step() StepInfo {
	return StepInfo{Name: "docker:" + d.Tags[0]}
}

func (d Docker) Run(ctx context.Context) error {
	b := d.Bin
	if b == "" {
		b = "docker"
	}
	if d.Context == "" {
		d.Context = "."
	}
	arg := []string{
		"build", "-f", d.File,
	}
	for _, tag := range d.Tags {
		arg = append(arg, "-t", tag)
	}
	arg = append(arg, d.Context)
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "create cluster")
	}
	return nil
}
