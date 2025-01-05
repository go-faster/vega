package installer

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

// Kind is Kubernetes In Docker (KIND) installer.
type Kind struct {
	Bin        string
	Name       string
	Config     string
	KubeConfig string
}

func (k Kind) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = "kind"
	}
	if k.Name == "" {
		k.Name = "vega"
	}
	{
		// Check for existing cluster.
		cmd := exec.CommandContext(ctx, b, "get", "clusters")
		buf := &bytes.Buffer{}
		cmd.Stdout = buf
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return errors.Wrap(err, "get clusters")
		}
		if bytes.Contains(buf.Bytes(), []byte(k.Name)) {
			// Delete existing cluster.
			cmd := exec.CommandContext(ctx, b, "delete", "cluster", "--name", k.Name)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return errors.Wrap(err, "delete cluster")
			}
		}
	}
	arg := []string{
		"create", "cluster",
		"-n", k.Name,
	}
	if k.Config != "" {
		arg = append(arg, "--config", k.Config)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if k.KubeConfig != "" {
		cmd.Env = append(cmd.Env, "KUBECONFIG="+k.KubeConfig)
	}
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "create cluster")
	}
	return nil
}

func (k Kind) Step() StepInfo {
	return StepInfo{Name: "kind"}
}

type KindLoad struct {
	Bin        string
	Name       string
	Images     []string
	KubeConfig string
}

func (k KindLoad) Step() StepInfo {
	return StepInfo{Name: "kind load"}
}

func (k KindLoad) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = "kind"
	}
	if k.Name == "" {
		k.Name = "vega"
	}
	arg := []string{
		"load", "docker-image",
		"--name", k.Name,
	}
	for _, img := range k.Images {
		arg = append(arg, img)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if k.KubeConfig != "" {
		cmd.Env = append(cmd.Env, "KUBECONFIG="+k.KubeConfig)
	}
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "load images")
	}
	return nil
}
