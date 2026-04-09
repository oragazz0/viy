# Chaos Primitives - "The Eyes"

Cada "eye" é um módulo independente de chaos que revela um tipo específico de fraqueza sistêmica.

---

## 1. Eye of Disintegration - Pod Deletion

**Conceito**: Revela dependência em instâncias específicas de pods

**Chaos Type**: Pod termination

**Configurações**:
```yaml
config:
  podKillCount: 3              # número fixo de pods
  # OU
  podKillPercentage: 30        # percentual do total
  interval: 60s                # tempo entre kills
  strategy: random             # random | sequential | worst-first
  gracePeriod: 0s              # immediate kill vs graceful
```

**Verdades Reveladas**:
- Pods são realmente stateless?
- Health checks estão configurados corretamente?
- Deployment controller está funcionando?
- Há single points of failure?

**Use Cases**:
- Testar recriação automática de pods
- Validar health checks e readiness probes
- Simular node failures

---

## 2. Eye of Charm - Network Chaos

**Conceito**: Revela dependências de rede e timeouts inadequados

**Chaos Type**: Network manipulation

**Configurações**:
```yaml
config:
  latency: 500ms
  jitter: 100ms                # variação aleatória
  packetLoss: 10%
  packetCorruption: 2%
  bandwidth: 1Mbps             # throttling
  affectInbound: true
  affectOutbound: true
  protocols: [tcp, udp]
```

**Implementação**:
- Linux `tc` (traffic control) via sidecars
- eBPF para packet manipulation (advanced)
- Istio/Linkerd integration (service mesh mode)

**Verdades Reveladas**:
- Timeouts estão configurados corretamente?
- Circuit breakers funcionam?
- Sistema degrada graciosamente em rede lenta?
- Retry logic é adequado?

**Use Cases**:
- Testar timeouts e retries
- Validar circuit breakers
- Simular redes instáveis/saturadas

---

## 3. Eye of Death - Resource Exhaustion ✅ IMPLEMENTED

**Conceito**: Revela problemas de resource limits e auto-scaling

**Chaos Type**: CPU/Memory/Disk stress

**Status**: Implementado em `pkg/eyes/death/`

**Configurações**:
```yaml
config:
  cpuStress: 80                # percentual de CPU load por worker (1-100)
  memoryStress: 70             # percentual de memória por worker (1-100)
  diskIOBytes: 1048576         # bytes por worker para disk I/O stress
  duration: 2m                 # duração do stress
  rampUp: 30s                  # aumento gradual (stress-ng --ramp-time)
  workers: 4                   # threads de stress
```

**Implementação**:
- Ephemeral containers com `stress-ng` (pod-level cgroup stress)
- Imagem: `alexeiled/stress-ng:latest` (hardcoded no MVP)
- Cleanup via `kill 1` exec no ephemeral container
- Partial failure: Unveil retorna sucesso mesmo se alguns pods falharem, surfacing via TruthsRevealed

**Notas de Implementação**:
- `diskIOStress` do spec original foi mudado para `diskIOBytes` (bytes por worker) porque stress-ng `--hdd-bytes` aceita valores absolutos, não porcentagens
- Ephemeral containers são append-only no K8s — o container permanece no pod spec mas o processo é terminado
- Requer K8s 1.25+ (ephemeral containers GA)

**Verdades Reveladas**:
- Resource limits estão configurados?
- HPA (Horizontal Pod Autoscaler) funciona?
- Aplicação lida bem com resource pressure?
- OOM killer comporta-se como esperado?

**Use Cases**:
- Testar auto-scaling (HPA)
- Validar resource limits/requests
- Simular noisy neighbors

---

## 4. Eye of Petrification - Container Freeze

**Conceito**: Revela dependências de timing e deadlocks

**Chaos Type**: Process suspension

**Configurações**:
```yaml
config:
  duration: 30s
  containers: [api, sidecar]   # específicos ou all
  signal: SIGSTOP              # ou custom
```

**Implementação**:
- `docker pause` / `crictl pause`
- SIGSTOP em processos específicos

**Verdades Reveladas**:
- Dependências detectam containers travados?
- Timeouts estão bem configurados?
- Sistema se recupera após unfreeze?

**Use Cases**:
- Simular deadlocks
- Testar timeouts de dependências
- Validar graceful degradation

---

## 5. Eye of Sleep - Latency Injection

**Conceito**: Revela impacto de slow dependencies

**Chaos Type**: Application-level delays

**Configurações**:
```yaml
config:
  delay: 2s
  endpoints: ["/api/users", "/api/orders"]
  probability: 50%             # % de requests afetadas
  distributionType: normal     # normal | uniform | exponential
```

**Implementação**:
- HTTP proxy interceptando requests
- Service mesh policy injection
- Application instrumentation (SDK)

**Verdades Reveladas**:
- UX degrada aceitavelmente?
- Cache funciona efetivamente?
- Database queries lentas impactam quanto?

**Use Cases**:
- Testar user experience degradada
- Validar cache effectiveness
- Simular slow database queries

---

## 6. Eye of Wounding - Partial Failures

**Conceito**: Revela problemas em error handling

**Chaos Type**: Intermittent errors

**Configurações**:
```yaml
config:
  errorRate: 20%               # % de requests com erro
  errorCodes: [500, 503, 504]
  pattern: intermittent        # intermittent | progressive | random
  affectedOperations: [read, write]
```

**Verdades Reveladas**:
- Retry logic está implementado?
- Error handling é adequado?
- Sistema tolera serviços flaky?

**Use Cases**:
- Testar retry logic
- Validar error handling
- Simular flaky services

---

## Eye Interface (Go)

```go
package eyes

import (
    "context"
    "time"
)

// Eye representa um módulo de chaos que revela verdades sobre a infraestrutura
type Eye interface {
    // Name retorna o identificador único do eye
    Name() string

    // Description retorna descrição human-readable
    Description() string

    // Unveil inicia a revelação através de chaos injection
    Unveil(ctx context.Context, target Target, config EyeConfig) error

    // Pause pausa temporariamente o chaos
    Pause(ctx context.Context) error

    // Close encerra o chaos e faz cleanup (fecha o olho)
    Close(ctx context.Context) error

    // Observe retorna métricas em tempo real (o que o olho vê)
    Observe() Metrics

    // Validate verifica se a configuração é válida
    Validate(config EyeConfig) error
}

// Target representa o alvo da revelação
type Target struct {
    Kind       string            // Pod, Deployment, Service, etc
    Name       string
    Namespace  string
    Labels     map[string]string
    Selector   string            // label selector
}

// EyeConfig é a configuração específica de cada eye
type EyeConfig interface {
    Validate() error
}

// Metrics representa o que um eye observa/revela
type Metrics struct {
    TargetsAffected   int
    OperationsTotal   int64
    ErrorsTotal       int64
    TruthsRevealed    []string  // Insights descobertos
    LastExecutionTime time.Time
    IsActive          bool
}
```
