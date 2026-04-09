# Configuração YAML

## Schema Completo

```yaml
apiVersion: chaos.viy.io/v1alpha1
kind: ChaosExperiment

metadata:
  name: api-resilience-revelation
  annotations:
    description: "Unveil weaknesses in API layer"
    owner: "sre-team@company.com"
    runbook: "https://wiki.company.com/chaos/api-test"

spec:
  # Eyes a serem abertos
  eyes:
    - type: disintegration
      target:
        kind: Deployment
        name: api-server
        namespace: production
        labels:
          app: api
          version: v2
      config:
        podKillCount: 3
        interval: 60s
        strategy: random
        gracePeriod: 30s

    - type: charm
      target:
        kind: Service
        name: database
      config:
        latency: 500ms
        jitter: 100ms
        packetLoss: 10%
        corruption: 2%
        bandwidth: 10Mbps
        affectInbound: true
        affectOutbound: false
        protocols: [tcp]

    - type: death
      target:
        kind: Pod
        selector:
          matchLabels:
            component: worker
      config:
        cpuStress: 80
        memoryStress: 70
        diskIOBytes: 1048576
        duration: 2m
        rampUp: 30s
        workers: 4

    - type: petrification
      target:
        kind: Deployment
        name: background-job
      config:
        duration: 45s
        containers: [job-processor]
        signal: SIGSTOP

    - type: sleep
      target:
        kind: Service
        name: api-gateway
      config:
        delay: 2s
        endpoints: ["/api/users", "/api/orders"]
        probability: 50%
        distributionType: normal

    - type: wounding
      target:
        kind: Service
        name: payment-service
      config:
        errorRate: 20%
        errorCodes: [500, 503, 504]
        pattern: intermittent
        affectedOperations: [read, write]

  # Controles de segurança (pálpebras)
  safety:
    maxBlastRadius: 50%
    respectPDB: true
    minHealthyReplicas: 2
    autoRollback: true
    rollbackTriggers:
      - type: errorRate
        threshold: 10%
        window: 1m
      - type: latencyP99
        threshold: 5s
        window: 1m
      - type: crashLoopBackoff
        count: 3

  # Duração da revelação
  duration: 10m
  # schedule: "*/30 * * * *"  # Cron (future feature)

  # Observabilidade (visão onisciente)
  observability:
    metrics:
      enabled: true
      port: 9090
      path: /metrics
    tracing:
      enabled: true
      exporter: otlp
      endpoint: "http://jaeger:4317"
      samplingRate: 1.0
    logging:
      level: info
      format: json
    notifications:
      slack:
        webhook: "https://hooks.slack.com/services/XXX"
        channel: "#sre-chaos"
        events: [started, completed, failed, rollback]
      pagerduty:
        routingKey: "R0XXXXXXXXXXXXXXXX"
        severity: warning

  # Pré-condições (antes de abrir os olhos)
  preconditions:
    - type: healthcheck
      endpoint: "http://api-server.production.svc:8080/health"
      expectedStatus: 200
      timeout: 5s
    - type: metricThreshold
      query: 'sum(rate(http_requests_total{namespace="production"}[5m]))'
      operator: ">"
      value: 100
      description: "Only unveil if traffic > 100 req/s"
```

---

## Validação de Schema

```go
func ValidateExperimentConfig(config *ChaosExperiment) error {
    // Validar versão da API
    if config.APIVersion != "chaos.viy.io/v1alpha1" {
        return fmt.Errorf("unsupported API version: %s", config.APIVersion)
    }

    // Validar eyes
    if len(config.Spec.Eyes) == 0 {
        return errors.New("at least one eye must be specified for revelation")
    }

    // Validar blast radius
    if config.Spec.Safety.MaxBlastRadius < 0 || config.Spec.Safety.MaxBlastRadius > 100 {
        return errors.New("maxBlastRadius must be between 0 and 100")
    }

    // Validar targets
    for _, eye := range config.Spec.Eyes {
        if eye.Target.Kind == "" || eye.Target.Name == "" {
            return fmt.Errorf("eye %s: target kind and name are required", eye.Type)
        }
    }

    return nil
}
```
