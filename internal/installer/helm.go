package installer

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

type HelmUpgrade struct {
	Bin             string
	Install         bool
	Values          string
	Name            string
	Chart           string
	Namespace       string
	CreateNamespace bool
	Version         string
	KubeConfig      string
}

func (h HelmUpgrade) Step() StepInfo {
	return StepInfo{Name: "helm upgrade: " + h.Name}
}

func (h HelmUpgrade) Run(ctx context.Context) error {
	b := h.Bin
	if b == "" {
		b = "helm"
	}
	arg := []string{
		"upgrade",
		h.Name, h.Chart,
	}
	if h.KubeConfig != "" {
		arg = append(arg, "--kubeconfig", h.KubeConfig)
	}
	if h.Install {
		arg = append(arg, "--install")
	}
	if h.Values != "" {
		arg = append(arg, "--values", h.Values)
	}
	if h.Namespace != "" {
		arg = append(arg, "--namespace", h.Namespace)
	}
	if h.CreateNamespace {
		arg = append(arg, "--create-namespace")
	}
	if h.Version != "" {
		arg = append(arg, "--version", h.Version)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "helm upgrade")
	}
	return nil
}
