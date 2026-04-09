# MVP Roadmap

## v0.1.0 - "First Unveiling" (2-3 semanas)

**Objetivo**: CLI funcional com chaos básico - abrir o primeiro olho

**Tasks**:
- [x] Setup projeto Go (estrutura, go.mod, Makefile)
- [x] Implementar CLI com Cobra (comandos básicos)
- [x] K8s client wrapper (connection, pod operations)
- [x] Eye interface + registry
- [x] Eye of Disintegration (pod kill apenas)
  - [x] Random pod selection
  - [x] Graceful termination
  - [x] Target resolution por label selector
- [x] Dry-run mode (`viy dream`)
- [x] JSON logging estruturado (zap)
- [x] State persistence (local file ~/.viy/state.json)
- [x] README básico com quick start + Viy lore
- [x] GitHub Actions CI (lint + test)

**Deliverables**:
```bash
# Deve funcionar:
viy dream --eye disintegration --target deployment/nginx
viy unveil --eye disintegration --target deployment/nginx --duration 2m
viy slumber --all
```

---

## v0.2.0 - "Multiple Eyes" (1-2 semanas)

**Objetivo**: Múltiplos chaos types e orquestração - despertar vários olhos

**Tasks**:
- [ ] Eye of Charm (network latency injection)
  - [ ] TC (traffic control) integration
  - [ ] Sidecar injection strategy
- [x] Eye of Death (CPU/memory/disk stress)
  - [x] stress-ng integration via ephemeral containers
  - [ ] Resource monitoring
- [ ] Multi-eye execution (`viy awaken`)
  - [ ] Concurrent execution com errgroup
  - [ ] Context propagation
- [ ] Prometheus metrics export
  - [ ] Basic metrics (operations, targets, truths_revealed)
  - [ ] HTTP server para scraping
- [ ] Blast radius controls
  - [ ] Percentage-based limiting
  - [ ] PDB awareness
- [ ] RBAC validation
  - [ ] Permission checking antes de executar
  - [ ] Error messages melhorados

**Deliverables**:
```bash
# Deve funcionar:
viy awaken --eyes disintegration,charm --target namespace/staging
curl localhost:9090/metrics | grep viy_
```

---

## v0.3.0 - "Omniscient Vision" (2 semanas)

**Objetivo**: Observabilidade completa - ver todas as verdades

**Tasks**:
- [ ] TUI dashboard (`viy scry`)
  - [ ] Real-time metrics display
  - [ ] Event log tail
  - [ ] Interactive controls (pause/stop)
  - [ ] "Truths Revealed" section
- [ ] Relatórios detalhados (`viy reveal`)
  - [ ] Markdown format
  - [ ] Impact analysis (before/during/after)
  - [ ] Timeline generation
  - [ ] Truths/insights section
- [ ] Auto-rollback mechanism
  - [ ] TTL-based
  - [ ] Threshold-based (error rate, latency)
  - [ ] Health check integration
- [ ] YAML configuration support
  - [ ] Schema definition (chaos.viy.io/v1alpha1)
  - [ ] Validation
  - [ ] Examples
- [ ] Enhanced logging
  - [ ] Structured fields
  - [ ] Log levels
  - [ ] File output option

**Deliverables**:
```bash
# Deve funcionar:
viy awaken -f experiment.yaml
viy scry --follow
viy reveal --experiment exp-123 --format markdown > revelation.md
```

---

## v0.4.0 - "Apocalypse Unveiled" (1 semana)

**Objetivo**: Features avançadas - revelação total

**Tasks**:
- [ ] Modo `--apocalypse`
  - [ ] Confirmation flow ("UNVEIL ALL")
  - [ ] ASCII art (Viy totalmente desperto)
  - [ ] All-eyes orchestration
  - [ ] Survival score calculation
- [ ] Eyes restantes:
  - [ ] Eye of Petrification (container freeze)
  - [ ] Eye of Sleep (latency injection)
  - [ ] Eye of Wounding (error injection)
- [ ] OpenTelemetry integration
  - [ ] Distributed tracing
  - [ ] Span creation
  - [ ] Context propagation
- [ ] Webhooks/Notifications
  - [ ] Slack integration
  - [ ] PagerDuty integration
  - [ ] Custom webhooks

**Deliverables**:
```bash
# Deve funcionar:
viy unveil --apocalypse --target namespace/staging --confirm-chaos
# Ver traces em Jaeger
```

---

## v1.0.0 - "Production Truth" (2-3 semanas de polish)

**Objetivo**: Production-ready release - verdade em produção

**Tasks**:
- [ ] Comprehensive documentation
  - [ ] Architecture diagrams
  - [ ] Philosophy doc (Viy lore + chaos engineering)
  - [ ] Eye-by-eye guides
  - [ ] Best practices
  - [ ] Troubleshooting
- [ ] Security hardening
  - [ ] RBAC ClusterRole minimal
  - [ ] Secret management
  - [ ] Audit logging
- [ ] Helm chart
  - [ ] Deployment templates
  - [ ] ConfigMap defaults
  - [ ] RBAC automation
- [ ] Performance optimization
  - [ ] Profiling (pprof)
  - [ ] Memory optimization
  - [ ] Binary size reduction (upx)
- [ ] End-to-end tests
  - [ ] Kind cluster automation
  - [ ] Smoke tests
- [ ] Release automation
  - [ ] GoReleaser config
  - [ ] GitHub releases
  - [ ] Homebrew formula (opcional)
- [ ] Marketing materials
  - [ ] Blog post
  - [ ] Demo video
  - [ ] Twitter/Reddit announcements

**Deliverables**:
- GitHub release com binaries para Linux/macOS/Windows
- Helm chart publicado
- Documentação completa no GitHub Pages
- Blog post lançamento
