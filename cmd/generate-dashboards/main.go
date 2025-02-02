package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-faster/errors"
	"sigs.k8s.io/yaml"
)

type Metadata struct {
	Name      string            `yaml:"name" json:"name,omitempty"`
	Namespace string            `yaml:"namespace" json:"namespace,omitempty"`
	Labels    map[string]string `yaml:"labels" json:"labels,omitempty"`
}
type ConfigMap struct {
	Kind       string            `yaml:"kind" json:"kind,omitempty"`
	APIVersion string            `yaml:"apiVersion" json:"apiVersion,omitempty"`
	Metadata   Metadata          `yaml:"metadata" json:"metadata"`
	Data       map[string]string `yaml:"data" json:"data,omitempty"`
}

func run() error {
	var arg struct {
		OutputFile string
	}
	flag.StringVar(&arg.OutputFile, "o", "", "Output file")
	flag.Parse()

	if arg.OutputFile == "" {
		return errors.New("output file is required")
	}

	f, err := os.Create(arg.OutputFile)
	if err != nil {
		return errors.Wrap(err, "os.Create")
	}
	defer func() {
		_ = f.Close()
	}()

	entries, err := os.ReadDir(filepath.Join("_hack", "dashboards"))
	if err != nil {
		return errors.Wrap(err, "os.ReadDir")
	}
	for _, entry := range entries {
		_, _ = fmt.Fprintf(f, "---\n")

		data, err := os.ReadFile(filepath.Join("_hack", "dashboards", entry.Name()))
		if err != nil {
			return errors.Wrap(err, "os.ReadFile")
		}
		cfg := ConfigMap{
			Kind:       "ConfigMap",
			APIVersion: "v1",
			Data: map[string]string{
				entry.Name(): string(data),
			},
			Metadata: Metadata{
				Name:      "vega-dashboards-" + entry.Name(),
				Namespace: "monitoring",
				Labels: map[string]string{
					"grafana_dashboard": "1",
				},
			},
		}
		cfg.Data[entry.Name()] = string(data)
		yamlData, err := yaml.Marshal(cfg)
		if err != nil {
			return errors.Wrap(nil, "yaml.Marshal")
		}
		_, _ = fmt.Fprint(f, string(yamlData))
	}

	if err := f.Close(); err != nil {
		return errors.Wrap(err, "f.Close")
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	}
}
