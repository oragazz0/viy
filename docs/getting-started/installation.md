# Installation

> Build Viy from source and install the binary.

## Prerequisites

- **Go 1.25+** — [download](https://go.dev/dl/)
- **kubectl** configured with access to a Kubernetes cluster (for running experiments)

## Build from Source

```bash
git clone https://github.com/oragazz0/viy.git
cd viy
go build -o viy ./cmd/viy
```

Move the binary to your `$PATH`:

```bash
sudo mv viy /usr/local/bin/
```

## Build with Version Info

Inject version metadata at build time using `-ldflags`:

```bash
go build -ldflags="\
  -X github.com/oragazz0/viy/internal/version.Version=0.1.0 \
  -X github.com/oragazz0/viy/internal/version.Commit=$(git rev-parse --short HEAD) \
  -X github.com/oragazz0/viy/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o viy ./cmd/viy
```

Verify the build:

```bash
viy version
```

Output:

```
👁️  Viy
  Version: 0.1.0
  Commit:  a1b2c3d
  Built:   2026-04-07T12:00:00Z
```

## Kubeconfig

Viy resolves Kubernetes credentials in the following order:

1. `--kubeconfig` flag (explicit path)
2. `$KUBECONFIG` environment variable or `~/.kube/config` (client-go defaults)
3. In-cluster service account (when running inside a pod)

## See Also

- [Quickstart](quickstart.md) — run your first experiment in 5 minutes
- [kind Setup](kind-setup.md) — set up a local cluster for safe testing
