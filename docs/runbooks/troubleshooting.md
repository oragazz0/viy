# Troubleshooting

> Common issues and how to resolve them.

## Connection Errors

### "building k8s config: ..."

Viy cannot connect to a Kubernetes cluster.

**Causes:**
- No kubeconfig file found
- Cluster is unreachable
- Invalid credentials

**Fix:**

```bash
# Verify kubectl works
kubectl cluster-info

# Explicitly point Viy to your kubeconfig
viy unveil --kubeconfig ~/.kube/config --eye disintegration --target deployment/nginx
```

If using kind:

```bash
kind get kubeconfig --name viy-test > /tmp/viy-kubeconfig
viy unveil --kubeconfig /tmp/viy-kubeconfig ...
```

## Safety Errors

### "namespace X is protected"

You tried to target `kube-system`, `kube-public`, or `kube-node-lease`. These namespaces are permanently blocked.

**Fix:** Target a different namespace. There is no override flag — this is a hard-coded safety constraint.

### "blast radius would be exceeded"

The combination of blast radius percentage and min healthy replicas prevents the experiment.

**Example:**

```
blast radius would be exceeded: 5 targets minus 2 affected leaves 3, below minimum 4
```

**Fix:**

```bash
# Option A: Increase blast radius
viy unveil ... --blast-radius 50%

# Option B: Lower min healthy
viy unveil ... --min-healthy 1

# Option C: Reduce kill count
viy unveil ... --config "podKillCount=1"
```

Use `dream` to preview before changing:

```bash
viy dream --eye disintegration --target deployment/nginx --blast-radius 50%
```

### "insufficient targets for requested kill count"

You requested more pod kills than available pods.

**Fix:** Lower `podKillCount` or deploy more replicas. Check current pod count:

```bash
kubectl get pods -l app=nginx --no-headers | wc -l
```

## Configuration Errors

### "invalid configuration: must specify podKillCount or podKillPercentage"

The eye config is missing a kill target. This happens when `--config` contains unrecognized keys only.

**Fix:** Ensure the config string contains valid keys:

```bash
# Wrong
viy unveil ... --config "count=3"

# Correct
viy unveil ... --config "podKillCount=3"
```

### "invalid configuration: cannot specify both podKillCount and podKillPercentage"

Both values were provided in the config string.

**Fix:** Use one or the other:

```bash
--config "podKillCount=3"
# OR
--config "podKillPercentage=50"
```

### "invalid log level"

The `--log-level` flag received an unsupported value.

**Fix:** Use one of: `debug`, `info`, `warn`, `error`.

## State Issues

### "No experiments found. Viy sleeps."

The state file has no experiments or doesn't exist yet. This is normal before running your first experiment.

### Stale Experiments in `viy vision`

Old experiments accumulate in `~/.viy/state.json`. There is no automatic cleanup.

**Fix:** Delete the state file to clear history:

```bash
rm ~/.viy/state.json
```

This is safe — it only removes experiment history, not running experiments.

### `viy slumber` Doesn't Stop a Running Experiment

**Known limitation.** Slumber updates the state file but cannot signal a running `viy unveil` process.

**Fix:** Send a signal directly to the running process:

```bash
# Find the process
ps aux | grep "viy unveil"

# Send SIGINT (graceful stop)
kill -INT <pid>
```

Viy handles SIGINT/SIGTERM gracefully — it will stop after the current operation and persist final state.

## Error Messages with Suggestions

Viy wraps some errors with actionable suggestions. Look for the format:

```
error message

💡 Suggestion: what to do about it
```

These suggestions come from `pkg/errors.WithSuggestion()` and point to concrete next steps.

## See Also

- [Safety Guards](../safety/guards.md) — blast radius and namespace protection details
- [State Persistence](../architecture/state.md) — how the state file works
- [CLI Commands](../cli/commands.md) — flag reference
