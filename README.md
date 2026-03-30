# 👁️ Viy

```
══════════════════════════════════════════════════════════════
  ⬡       ⬡       ⬡
   ╲      │      ╱
     ▄████████████▄
    █░░╱ ▓▒◉▒▓ ╲░░█       V I Y
     ▀████████████▀
   ╱      │      ╲        Kubernetes Chaos Engineering Toolkit
  ⬡       ⬡       ⬡
══════════════════════════════════════════════════════════════
```

**Viy** is a Kubernetes Chaos Engineering CLI toolkit written in Go.

Inspired by the creature from Slavic folklore whose gaze reveals absolute truth, Viy "opens its eyes" on your infrastructure — exposing hidden weaknesses through controlled chaos and omniscient observability.

> *"I summon the vampires! I summon the werewolves!... I summon Viy!"*

---

## Key Concepts

| Folklore | Technical Equivalent |
|----------|---------------------|
| **Viy's massive eye** | Core orchestration engine |
| **Servants lifting eyelids** | Chaos modules (*eyes*) |
| **Revealing gaze** | Full observability during experiments |
| **Fatal truth** | Failures that must be fixed |
| **Heavy eyelids** | Safety guards against uncontrolled chaos |

Each chaos module is called an **Eye**. Each Eye reveals a different kind of systemic
weakness:

| Eye | Chaos Type | What It Reveals |
|-----|-----------|-----------------|
| **Disintegration** *(v0.1)* | Pod deletion | Auto-recovery, health checks, single points of failure |
| **Charm** *(v0.2)* | Network manipulation | Timeout gaps, circuit breaker failures, retry logic |
| **Death** *(v0.2)* | CPU/Memory stress | Resource limits, HPA, noisy neighbor tolerance |
| **Petrification** *(v0.4)* | Container freeze | Deadlocks, dependency timeouts |
| **Sleep** *(v0.4)* | Latency injection | UX degradation, cache effectiveness |
| **Wounding** *(v0.4)* | Error injection | Error handling gaps, flaky service tolerance |

---

## Quick Start

### Install

```bash
# Build from source
git clone https://github.com/oragazz0/viy.git
cd viy
make build

# Binary is at ./viy
./viy version
```

### First Revelation (Dry-Run)

Always dream before unveiling:

```bash
viy dream \
  --eye disintegration \
  --target deployment/nginx \
  --namespace staging
```

Output:

```
🔮 Dream Mode: Viy dreams of revelation...

Targets that would be unveiled:
  • Pod: nginx-abc123 (staging/nginx)
  • Pod: nginx-def456 (staging/nginx)
  • Pod: nginx-ghi789 (staging/nginx)

Estimated blast radius: 30% (3/10 pods)
Safety checks: ✅ All passed

💡 Remove --dream flag to unveil for real.
```

### Unveil for Real

```bash
viy unveil \
  --eye disintegration \
  --target deployment/nginx \
  --namespace staging \
  --duration 2m \
  --blast-radius 30% \
  --min-healthy 2
```

Output:

```
👁️  Viy awakens from slumber...

🎯 Target Resolution
   ✅ Found 10 pods in deployment/nginx
   ✅ Blast radius: 3/10 pods (30%)
   ✅ Safety checks passed

🔮 Opening Eye of Disintegration
   ⚡ Unveiling pod nginx-abc123... truth revealed
   ⚡ Unveiling pod nginx-def456... truth revealed
   ⚡ Unveiling pod nginx-ghi789... truth revealed

💡 Truths Revealed:
   🟢 Pod auto-recovery working correctly
```

### Check Active Experiments

```bash
viy vision --all
```

```
ID            EYES               STATUS      TARGET              STARTED
a1b2c3d4e5f6  [disintegration]   revealed    staging/nginx       2m ago
```

### Stop Everything

```bash
viy slumber --all
```

---

## CLI Reference

```
viy
├── unveil    Start experiment (open one eye)
├── dream     Dry-run mode (dream without executing)
├── vision    List active/past experiments
├── slumber   Stop experiments (close eyes)
└── version   Show version info
```

### `viy unveil`

| Flag | Default | Description |
|------|---------|-------------|
| `--eye` | *(required)* | Eye to open: `disintegration` |
| `--target` | *(required)* | K8s resource, e.g. `deployment/nginx` |
| `--namespace` | `default` | Kubernetes namespace |
| `--duration` | `5m` | How long the revelation lasts |
| `--blast-radius` | `30%` | Max % of targets to affect (1–100) |
| `--min-healthy` | `1` | Minimum healthy replicas to preserve |
| `--config` | | Eye config as `key=value,key=value` |
| `--dream` | `false` | Run in dry-run mode |

### Disintegration Config Keys

| Key | Example | Description |
|-----|---------|-------------|
| `podKillCount` | `3` | Fixed number of pods to kill |
| `podKillPercentage` | `30` | Percentage of pods to kill |
| `interval` | `60s` | Wait between kills |
| `strategy` | `random` | `random` or `sequential` |
| `gracePeriod` | `30s` | Grace period before termination |

```bash
viy unveil \
  --eye disintegration \
  --target deployment/api \
  --config "podKillPercentage=30,interval=60s,strategy=random,gracePeriod=30s" \
  --duration 10m
```

---

## Safety

Viy reveals truths, but responsibly. The heavy eyelids are safety guards:

- **Protected namespaces** — `kube-system`, `kube-public`, and `kube-node-lease` are
  blocked by default
- **Blast radius** — limits the percentage of targets affected (default: 30%)
- **Minimum healthy replicas** — never drops below `--min-healthy` (default: 1)
- **Dry-run** — always `dream` before you `unveil`
- **Signal handling** — Ctrl+C gracefully stops experiments
- **State persistence** — experiments are tracked at `~/.viy/state.json` with atomic writes

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                CLI Interface (Cobra)                     │
│  viy unveil | dream | vision | slumber | version         │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│           Core Orchestrator (Viy's Eye)                  │
│  • Target Resolution  • Safety Checks  • Lifecycle Mgmt │
└─────────┬────────────────────────────────────┬──────────┘
          │                                    │
┌─────────▼─────────┐              ┌──────────▼──────────┐
│   Eye Registry    │              │  Observability      │
│  • Discovery      │              │  • Structured JSON  │
│  • Validation     │              │    Logging (zap)    │
│  • Lifecycle      │              └─────────────────────┘
└─────────┬─────────┘
          │
┌─────────▼──────────────────────────────────────────────┐
│              Eyes (Chaos Modules)                        │
├─────────────────────────────────────────────────────────┤
│ Disintegration (v0.1)  │  Charm, Death, ... (planned)   │
└────┬────────────────────────────────────────────────────┘
     │
┌────▼────────────────────────────────────────────────────┐
│           Kubernetes API (client-go)                     │
└─────────────────────────────────────────────────────────┘
```

### Package Layout

```
viy/
├── cmd/viy/main.go                          Entry point
├── pkg/
│   ├── cli/                                 Cobra commands
│   ├── eyes/                                Eye interface + registry
│   │   └── disintegration/                  Pod kill module
│   ├── errors/                              Sentinel errors
│   ├── k8s/                                 Kubernetes client wrapper
│   ├── observability/                       Structured logging
│   ├── orchestrator/                        Core engine
│   ├── safety/                              Blast radius calculator
│   └── version/                             Build info
└── internal/
    └── state/                               Local state persistence
```

---

## Development

```bash
make build      # Build binary with version injection
make test       # Run tests with -race
make lint       # golangci-lint
make vuln       # Vulnerability scan
make clean      # Remove binary
```

### Testing

```bash
# All tests
go test -v -race -count=1 ./...

# Specific package
go test -v ./internal/eyes/disintegration/...
```

---

## Roadmap

- [x] **v0.1.0** — Eye of Disintegration, CLI, dry-run, state persistence
- [ ] **v0.2.0** — Eye of Charm (network), Eye of Death (resources), multi-eye, Prometheus
- [ ] **v0.3.0** — TUI dashboard (`viy scry`), reports (`viy reveal`), YAML configs
- [ ] **v0.4.0** — Apocalypse mode, remaining Eyes, OpenTelemetry
- [ ] **v1.0.0** — Production hardening, Helm chart, docs, release

---

## Philosophy

Viy is not just a chaos tool — it is a **truth revealer**.

Traditional chaos engineering says *"break things to see what happens."*
Viy says *"open your eyes to see what was always there."*

Every experiment is a **revelation**. Every failure is a **truth** your infrastructure was hiding.
The goal is not destruction — it is **understanding**.

> *"A curse upon you! With the wings of a bat! With the blood of a serpent! I shall curse you! Curse you!"*

---

## License

MIT
