package installer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-faster/errors"
)

// GoBuild builds go binary.
type GoBuild struct {
	// Binary from "cmd" to build. For example, "vega-agent".
	Binary string
}

// BuildBinary is constructor for [GoBuild].
func BuildBinary(name string) GoBuild {
	return GoBuild{Binary: name}
}

// Step implements [Step].
func (g GoBuild) Step() StepInfo {
	return StepInfo{
		Name: fmt.Sprintf("go(%s)", g.Binary),
	}
}

// Run a go build.
//
// #nosec: G204
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

	env := os.Environ()
	env = append(env, "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "exec go (stderr: %s, stdout: %s)", stderr, stdout)
	}

	return nil
}
