package k8s

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/go-faster/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/homedir"
)

// DefaultConfigPath returns kubeconfig path that can be used as default
// for command line flag.
//
// Will return an empty string if no KUBECONFIG is set and default config
// does not exist.
func DefaultConfigPath() string {
	defaultCfgPath := os.Getenv("KUBECONFIG")
	if home := homedir.HomeDir(); home != "" && defaultCfgPath == "" {
		p := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(p); err == nil {
			defaultCfgPath = p
		}
	}

	return defaultCfgPath
}

type Options struct {
	Path    string // if blank, in-cluster config will be used
	Context string // context name, optional
}

func OptionsFromFlags() *Options {
	var opt Options
	flag.StringVar(&opt.Path, "kubeconfig", DefaultConfigPath(), "absolute path to the kubeconfig file")
	flag.StringVar(&opt.Context, "context", "", "kubeconfig context to use")
	return &opt
}

func (o *Options) Config() (*rest.Config, error) {
	if o == nil || o.Path == "" {
		return rest.InClusterConfig()
	}
	rules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: o.Path,
	}
	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: o.Context,
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	// HACK: ease rate limits
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 500)

	return cfg, nil
}

type RemoteCluster struct {
	Server string // https://host:port
	Token  string
	CA     string // certificate authority data
}

func (c *RemoteCluster) Config() (*rest.Config, error) {
	cfg, err := clientcmd.NewDefaultClientConfig(clientcmdapi.Config{
		CurrentContext: "context",
		Contexts: map[string]*clientcmdapi.Context{
			"context": {
				Cluster:  "cluster",
				AuthInfo: "user",
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"user": {
				Token: c.Token,
			},
		},
		Clusters: map[string]*clientcmdapi.Cluster{
			"cluster": {
				Server:                   c.Server,
				CertificateAuthorityData: []byte(c.CA),
			},
		},
	}, nil).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create default client config")
	}

	// HACK: ease rate limits
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 500)

	return cfg, nil
}
