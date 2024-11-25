package installer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/go-faster/errors"
)

type GoBuild struct {
	// Binary from "cmd" to build. For example, "vega-agent".
	Binary string
}

func (g GoBuild) Step() StepInfo {
	return StepInfo{
		Name: fmt.Sprintf("go(%s)", g.Binary),
	}
}

// Run a go build.
func (g GoBuild) Run(ctx context.Context) error {
	args := []string{
		"build",
		"-o", filepath.Join("_out", "bin", g.Binary),
		fmt.Sprintf("./cmd/%s", g.Binary),
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	stderr := bytes.NewBuffer(nil)
	stdout := bytes.NewBuffer(nil)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "exec go (stderr: %s, stdout: %s)", stderr, stdout)
	}

	return nil
}
