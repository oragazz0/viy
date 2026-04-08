# Quickstart

> Run your first chaos experiment in 5 minutes.

## 1. Deploy a Sample Workload

Create a simple nginx deployment with 5 replicas:

```bash
kubectl create deployment nginx --image=nginx --replicas=5
```

Wait for all pods to be ready:

```bash
kubectl wait --for=condition=ready pod -l app=nginx --timeout=60s
```

## 2. Dream First (Dry Run)

Before running real chaos, use `dream` to preview what would happen:

```bash
viy dream --eye disintegration --target deployment/nginx
```

Output:

```
🔮 Dream Mode: Viy dreams of revelation...

Targets that would be unveiled:
  • Pod: nginx-abc123 (default/nginx)

Estimated blast radius: 30% (1/5 pods)
Safety checks: ✅ All passed
```

No pods are harmed. This shows exactly which pods would be affected.

## 3. Unveil (Live Experiment)

Run the real experiment — kill 1 pod and observe Kubernetes recovery:

```bash
viy unveil \
  --eye disintegration \
  --target deployment/nginx \
  --duration 1m
```

Viy will:

1. Resolve pods matching `app=nginx` in the `default` namespace
2. Validate blast radius (30% by default, min 1 healthy replica)
3. Delete the selected pod
4. Track the experiment in `~/.viy/state.json`

## 4. Check Experiment Status

```bash
viy vision
```

Output:

```
ID            EYES               STATUS      TARGET          STARTED
a1b2c3d4e5f6  [disintegration]   revealed    default/nginx   45s ago
```

## 5. Verify Recovery

Check that Kubernetes replaced the deleted pod:

```bash
kubectl get pods -l app=nginx
```

You should see 5 pods running — the deleted one was replaced automatically.

## Next Steps

- Kill multiple pods: `--config "podKillCount=3"`
- Use percentage-based kills: `--config "podKillPercentage=50%"`
- Change selection strategy: `--config "strategy=sequential"`
- Tighten safety: `--blast-radius 20% --min-healthy 3`

## See Also

- [Eye of Disintegration](../eyes/disintegration.md) — full configuration reference
- [CLI Commands](../cli/commands.md) — all commands and flags
- [Safety Guards](../safety/guards.md) — how Viy protects your cluster
