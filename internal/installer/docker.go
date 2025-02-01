package installer

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/go-faster/errors"
	"golang.org/x/sync/errgroup"
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

type DockerPull struct {
	Images     []string
	ImagesFile string
}

func (d DockerPull) Step() StepInfo {
	return StepInfo{Name: "docker pull"}
}

func (d DockerPull) Run(ctx context.Context) error {
	if os.Getenv("GITHUB_ACTIONS") != "" {
		// Skip pull on GitHub Actions
		fmt.Println("> Skipped (in github actions)")
		return nil
	}

	images := d.Images

	if d.ImagesFile != "" {
		file, err := os.Open(d.ImagesFile)
		if err != nil {
			return errors.Wrap(err, "open images file")
		}
		defer func() {
			_ = file.Close()
		}()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			image := scanner.Text()
			if image != "" {
				images = append(images, image)
			}
		}

		if err := scanner.Err(); err != nil {
			return errors.Wrap(err, "scan images file")
		}
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, image := range images {
		g.Go(func() error {
			cmd := exec.CommandContext(ctx, "docker", "pull", image)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return errors.Wrapf(err, "pull image %s", image)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "pull images")
	}

	return nil
}
