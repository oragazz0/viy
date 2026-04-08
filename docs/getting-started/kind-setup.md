# Local Cluster with kind

> Set up a disposable Kubernetes cluster for safe chaos experimentation.

## Prerequisites

- **Docker** — [install](https://docs.docker.com/get-docker/)
- **kind** — [install](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)

## Create a Cluster

```bash
kind create cluster --name viy-test
```

Verify connectivity:

```bash
kubectl cluster-info --context kind-viy-test
```

## Deploy a Test Workload

Create a deployment with enough replicas to experiment safely:

```bash
kubectl create deployment web --image=nginx --replicas=10
kubectl wait --for=condition=ready pod -l app=web --timeout=120s
```

10 replicas gives room to test various blast radius configurations without running out of pods.

## Run Experiments

Preview with dream mode:

```bash
viy dream --eye disintegration --target deployment/web --config "podKillCount=3"
```

Run live:

```bash
viy unveil --eye disintegration --target deployment/web --config "podKillCount=3" --duration 2m
```

## Cleanup

Delete the cluster when done:

```bash
kind delete cluster --name viy-test
```

This removes the entire cluster and all workloads — no residual state on your machine except `~/.viy/state.json`.

## See Also

- [Quickstart](quickstart.md) — first experiment walkthrough
- [Troubleshooting](../runbooks/troubleshooting.md) — common issues and fixes
