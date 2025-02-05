package installer

import (
	"context"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
)

type KubeApply struct {
	Bin        string
	File       string
	KubeConfig string
}

const kubectlBin = "kubectl"

func (k KubeApply) Step() StepInfo {
	return StepInfo{Name: "kubectl apply -f " + k.File}
}

func (k KubeApply) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = kubectlBin
	}
	arg := []string{
		"apply", "-f", k.File,
	}
	if k.KubeConfig != "" {
		arg = append(arg, "--kubeconfig", k.KubeConfig)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl apply -f")
	}
	return nil
}

type KubeRestart struct {
	Bin        string
	Target     string
	Name       string
	Namespace  string
	KubeConfig string
}

func (k KubeRestart) Step() StepInfo {
	return StepInfo{Name: "kubectl rollout restart " + k.Target + "/" + k.Name}
}

func (k KubeRestart) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = kubectlBin
	}
	arg := []string{
		"rollout", "restart", k.Target + "/" + k.Name,
	}
	if k.Namespace != "" {
		arg = append(arg, "-n", k.Namespace)
	}
	if k.KubeConfig != "" {
		arg = append(arg, "--kubeconfig", k.KubeConfig)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl rollout restart")
	}
	return nil
}

type KubeCreate struct {
	Bin        string
	File       string
	KubeConfig string
}

func (k KubeCreate) Step() StepInfo {
	return StepInfo{Name: "kubectl create -f " + k.File}
}

func (k KubeCreate) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = kubectlBin
	}
	arg := []string{
		"create", "-f", k.File,
	}
	if k.KubeConfig != "" {
		arg = append(arg, "--kubeconfig", k.KubeConfig)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl create -f")
	}
	return nil
}

type KubeDelete struct {
	Bin        string
	File       string
	KubeConfig string
}

func (k KubeDelete) Step() StepInfo {
	return StepInfo{Name: "kubectl delete -f " + k.File}
}

func (k KubeDelete) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = kubectlBin
	}
	arg := []string{
		"delete", "--ignore-not-found=true", "-f", k.File,
	}
	if k.KubeConfig != "" {
		arg = append(arg, "--kubeconfig", k.KubeConfig)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl delete -f")
	}
	return nil
}

type KubeRolloutStatus struct {
	Bin        string
	Target     string
	Name       string
	Namespace  string
	Watch      bool
	KubeConfig string
}

func (k KubeRolloutStatus) Step() StepInfo {
	return StepInfo{Name: "kubectl rollout status " + k.Target + "/" + k.Name}
}

func (k KubeRolloutStatus) Run(ctx context.Context) error {
	b := k.Bin
	if b == "" {
		b = kubectlBin
	}
	arg := []string{
		"rollout", "status", k.Target + "/" + k.Name,
	}
	if k.Namespace != "" {
		arg = append(arg, "-n", k.Namespace)
	}
	if k.Watch {
		arg = append(arg, "-w")
	}
	if k.KubeConfig != "" {
		arg = append(arg, "--kubeconfig", k.KubeConfig)
	}
	cmd := exec.CommandContext(ctx, b, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "kubectl rollout status")
	}
	return nil
}
