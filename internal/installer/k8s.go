package installer

import (
	"context"

	"github.com/go-faster/errors"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
)

type DaemonSet struct {
	KubeConfig  string
	KubeContext string

	Name      string
	Namespace string
	Image     string
}

func (s DaemonSet) Step() StepInfo {
	return StepInfo{Name: "daemonset: " + s.Name}
}

func P[T any](t T) *T {
	return &t
}

func (s DaemonSet) Run(ctx context.Context) error {
	rules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: s.KubeConfig,
	}
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: s.KubeContext,
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return err
	}
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(1000, 1000)
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "k8s: new")
	}
	container := core.Container{
		Name:  s.Name,
		Image: s.Image,
	}
	containers := []core.Container{container}
	spec := core.PodSpec{
		TerminationGracePeriodSeconds: P(int64(30)),
		Tolerations: []core.Toleration{
			{
				Key:      "node-role.kubernetes.io/master",
				Operator: core.TolerationOpExists,
				Effect:   core.TaintEffectNoSchedule,
			},
		},
		Containers: containers,
	}
	labels := map[string]string{
		"app": s.Name,
	}
	selector := map[string]string{
		"app": s.Name,
	}
	ds := &apps.DaemonSet{
		TypeMeta: meta.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: meta.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
			Labels:    labels,
		},
		Spec: apps.DaemonSetSpec{
			Selector: &meta.LabelSelector{
				MatchLabels: selector,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: labels,
				},
				Spec: spec,
			},
		},
	}
	if _, err = client.AppsV1().DaemonSets(s.Namespace).Create(ctx, ds, meta.CreateOptions{}); err != nil {
		return errors.Wrap(err, "k8s: create daemonset")
	}
	return nil
}
