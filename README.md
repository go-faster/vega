# vega [![Go Reference](https://img.shields.io/badge/go-pkg-00ADD8)](https://pkg.go.dev/github.com/go-faster/vega#section-documentation) [![codecov](https://img.shields.io/codecov/c/github/go-faster/vega?label=cover)](https://codecov.io/gh/go-faster/vega) [![experimental](https://img.shields.io/badge/-experimental-blueviolet)](https://go-faster.org/docs/projects/status#experimental)

Work in progress.

Research, development and best practices incubator for:
- Platform engineering
- Application development
- Observability
- Monitoring
- Configuration management
- Documentation
- Integration and performance testing
- Integrations
  - Tetragon
  - Cilium
  - Hubble

## Running

Requirements:
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
- [helm](https://helm.sh/docs/intro/install/)
- [cilium cli](https://docs.cilium.io/en/stable/gettingstarted/k8s-install-default/#install-the-cilium-cli)
- docker
- go 1.24

Add to hosts:
```
127.0.0.1 gitlab.vega.svc.cluster.local
```

```bash
make helm
```

```bash
source activate.sh
```

```console
$ go install ./cmd/v
$ v list
name=simon.client       ns=simon
name=simon.server       ns=simon
name=vega.agent ns=vega
name=vega.api   ns=vega
name=vega.ingest        ns=vega
$ v get simon.server
simon.server (ns=simon)
pods:
  simon-server-9f69cf65d-lhwk8 (mem=30 MB, cpu=1.013640, rx=3.9 MB/s, tx=29 kB/s)
  simon-server-9f69cf65d-nb6jz (mem=27 MB, cpu=0.864854, rx=4.7 MB/s, tx=39 kB/s)
  simon-server-9f69cf65d-v2tg4 (mem=29 MB, cpu=0.860625, rx=4.7 MB/s, tx=34 kB/s)
$ v get simon.client
simon.client (ns=simon)
pods:
  simon-client-6b4947b797-2dn2n (mem=19 MB, cpu=0.282894, rx=2.5 kB/s, tx=994 kB/s)
  simon-client-6b4947b797-444dk (mem=19 MB, cpu=0.146285, rx=2.8 kB/s, tx=1.1 MB/s)
  simon-client-6b4947b797-4ftnq (mem=19 MB, cpu=0.267662, rx=4.4 kB/s, tx=1.2 MB/s)
  simon-client-6b4947b797-5fvd8 (mem=17 MB, cpu=0.116613, rx=2.0 kB/s, tx=1.1 MB/s)
  simon-client-6b4947b797-7fkwb (mem=18 MB, cpu=0.214541, rx=2.6 kB/s, tx=924 kB/s)
  simon-client-6b4947b797-82wv8 (mem=18 MB, cpu=0.132827, rx=4.3 kB/s, tx=1.1 MB/s)
  simon-client-6b4947b797-9vsll (mem=19 MB, cpu=0.267073, rx=2.3 kB/s, tx=783 kB/s)
  simon-client-6b4947b797-ffbhr (mem=18 MB, cpu=0.238145, rx=3.1 kB/s, tx=1.6 MB/s)
  simon-client-6b4947b797-jl45t (mem=17 MB, cpu=0.242046, rx=2.0 kB/s, tx=1.2 MB/s)
  simon-client-6b4947b797-qxzfp (mem=18 MB, cpu=0.238922, rx=2.4 kB/s, tx=924 kB/s)
  simon-client-6b4947b797-sf94d (mem=17 MB, cpu=0.105813, rx=1.9 kB/s, tx=676 kB/s)
  simon-client-6b4947b797-xsx7c (mem=17 MB, cpu=0.248820, rx=2.0 kB/s, tx=1.1 MB/s)
```
