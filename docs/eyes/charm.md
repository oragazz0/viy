# Eye of Charm

> Network chaos — reveals truths about network dependencies, timeouts, and circuit breaker behavior.

## Overview

The Eye of Charm injects controlled network degradation into pods via ephemeral containers running [tc netem](https://man7.org/linux/man-pages/man8/tc-netem.8.html) from [netshoot](https://github.com/nicolaka/netshoot). It answers questions like:

- Are timeouts configured correctly for downstream dependencies?
- Do circuit breakers trip under network degradation?
- Does the system degrade gracefully on a slow or lossy network?
- Is retry logic adequate and properly bounded?

## Mechanism

Charm injects a **netshoot ephemeral container** with `NET_ADMIN` capability into each target pod. Since all containers in a pod share the same network namespace, `tc` rules applied from the ephemeral container affect the pod's entire network stack.

**Lifecycle:**

1. Discover target pods via label selector
2. For each pod, inject an ephemeral container running `sleep infinity` (keeps it alive for exec)
3. Auto-detect the default network interface (or use the explicit override)
4. Exec `tc qdisc add dev <iface> root netem ...` to apply chaos rules
5. On Pause or Close, exec `tc qdisc del dev <iface> root` to remove all rules

**Partial failure handling:** If injection or tc application fails for some pods but succeeds for others, Unveil succeeds with a degraded result. The partial failure is surfaced as a truth in `Observe()` metrics.

> Note: Ephemeral containers are append-only in Kubernetes. The container entry remains on the pod spec after the experiment ends, but the `tc` rules are fully removed on Pause/Close.

## Configuration

Pass eye-specific config via `--config` as comma-separated `key=value` pairs.

| Key | Type | Default | Description |
|---|---|---|---|
| `latency` | duration | `0s` | Fixed delay added to each outgoing packet |
| `jitter` | duration | `0s` | Random variation added to latency (requires `latency`) |
| `packetLoss` | float (0-100) | `0` | Percentage of packets dropped |
| `corruption` | float (0-100) | `0` | Percentage of packets with random bit flips |
| `duration` | duration | — | How long the experiment runs (required) |
| `interface` | string | auto-detect | Network interface to target (e.g. `eth0`, `ens5`) |

### Validation Rules

- At least one chaos parameter must be set (`latency`, `packetLoss`, or `corruption` > 0)
- `latency` must be non-negative
- `jitter` must be non-negative and requires `latency` to be set
- `packetLoss` and `corruption` must be between 0% and 100%
- `duration` must be positive

### tc netem Command Mapping

The config maps to tc flags as follows:

| Config | tc netem Flags |
|---|---|
| `latency: 500ms` | `delay 500ms` |
| `latency: 500ms, jitter: 100ms` | `delay 500ms 100ms` |
| `packetLoss: 10` | `loss 10.00%` |
| `corruption: 2.5` | `corrupt 2.50%` |
| all combined | `delay 500ms 100ms loss 10.00% corrupt 2.50%` |

### Interface Auto-Detection

When `interface` is not set, Charm detects the default-route interface via:

```bash
ip -o route show default | awk '{print $5}' | head -n1
```

Falls back to `eth0` if detection fails. Use the explicit `interface` config to override when your CNI uses non-standard interface names (e.g. Calico's `cali*`).

## Examples

Latency injection (500ms delay with 100ms jitter):

```bash
viy unveil --eye charm --target deployment/api-server \
  --config "latency=500ms,jitter=100ms,duration=2m"
```

Packet loss only:

```bash
viy unveil --eye charm --target deployment/payment-service \
  --config "packetLoss=10,duration=5m"
```

Combined network degradation:

```bash
viy unveil --eye charm --target deployment/api-server \
  --config "latency=200ms,jitter=50ms,packetLoss=5,corruption=1,duration=3m"
```

Explicit interface override:

```bash
viy unveil --eye charm --target deployment/api-server \
  --config "latency=500ms,duration=2m,interface=ens5"
```

Preview without executing:

```bash
viy dream --eye charm --target deployment/api-server \
  --config "latency=500ms,packetLoss=10,duration=2m"
```

## Execution Flow

1. Resolve the target resource via the Kubernetes API
2. Extract the resource's pod selector, merge with any user-supplied `--selector`
3. List matching pods
4. For each pod:
   - Generate a unique ephemeral container name (`viy-charm-<pod-prefix>`)
   - Inject netshoot ephemeral container with `NET_ADMIN` capability
   - Auto-detect network interface (or use override)
   - Exec `tc qdisc add` with netem parameters
   - If any step fails, log the error and continue to next pod
   - Track successfully charmed pods for cleanup
5. Record truths: how many pods were charmed, whether any failed

## Cleanup

Cleanup is **critical** for the Eye of Charm. Unlike pod termination (Disintegration) or process stress (Death), failed cleanup leaves pods with permanently degraded networking.

On `Close` or `Pause`, Charm executes `tc qdisc del dev <iface> root` on every charmed pod. If cleanup fails for a pod (e.g. the ephemeral container was evicted), a warning is logged with the pod name and namespace so the operator can manually clean up.

Manual cleanup if needed:

```bash
kubectl exec -it <pod-name> -c viy-charm-<prefix> -- tc qdisc del dev eth0 root
```

## Metrics

During and after execution, `Observe()` returns:

| Metric | Description |
|---|---|
| `EyeName` | Always `"charm"` |
| `TargetsAffected` | Number of pods where tc rules were successfully applied |
| `OperationsTotal` | Total injection + exec API calls made |
| `ErrorsTotal` | Operations that failed (injection or tc exec) |
| `TruthsRevealed` | Summary of what was revealed, including partial failures |
| `IsActive` | Whether the eye is currently active |

## Error Handling

| Error | Cause | What to do |
|---|---|---|
| `target not found` | No pods match the selector | Check the target name, namespace, and selector |
| `invalid configuration` | Config validation failed | Check that at least one chaos parameter is set and values are in range |
| `failed to reveal network truths` | Kubernetes API error | Check cluster connectivity and RBAC permissions |
| `inject netshoot sidecar` | Ephemeral container injection failed | Check ephemeral container support and security policies |
| `apply tc netem rules` | tc exec failed inside the container | Check that `NET_ADMIN` capability is allowed by your PodSecurityPolicy/Standards |
| Partial failure (surfaced as truth) | Some pods failed | Check pod status and security context constraints |

## Prerequisites

- Kubernetes 1.25+ (ephemeral containers GA)
- The netshoot image (`nicolaka/netshoot:latest`) must be pullable from target nodes
- RBAC permissions to patch pods (ephemeral containers subresource)
- `NET_ADMIN` capability must be allowed by the cluster's security policy (PodSecurity admission, OPA/Gatekeeper, etc.)

## Current Limitations (Phase A)

- **Egress only** — tc netem on the root qdisc only affects outgoing traffic. Inbound traffic manipulation requires `ifb` device setup (planned for Phase B).
- **No protocol filtering** — chaos applies to all traffic on the interface. TCP/UDP-specific filtering via `tc filter` is planned for Phase B.
- **No bandwidth throttling** — requires `tc tbf` qdisc chaining (planned for Phase B).

## See Also

- [Eyes Overview](overview.md) — interface and registry
- [Eye of Death](death.md) — resource exhaustion (reference implementation for ephemeral containers)
- [Safety Guards](../safety/guards.md) — blast radius and min healthy constraints
- [CLI Commands](../cli/commands.md) — `unveil` and `dream` usage
