package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/oragazz0/viy/internal/k8s"
	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
)

// fakeEye is a controllable eye.Eye used by multi-orchestrator tests.
type fakeEye struct {
	name          string
	unveilStart   chan struct{}
	unveilFunc    func(ctx context.Context) error
	closeFunc     func(ctx context.Context) error
	validateFunc  func(cfg eyes.EyeConfig) error
	unveilStarted int32
	unveilDone    int32
	closeDone     int32
	startedAt     atomic.Int64
}

func newFakeEye(name string) *fakeEye {
	return &fakeEye{
		name:        name,
		unveilStart: make(chan struct{}, 1),
	}
}

func (f *fakeEye) Name() string        { return f.name }
func (f *fakeEye) Description() string { return "fake eye for testing" }

func (f *fakeEye) Validate(cfg eyes.EyeConfig) error {
	if f.validateFunc != nil {
		return f.validateFunc(cfg)
	}
	return nil
}

func (f *fakeEye) Unveil(ctx context.Context, _ eyes.Target, _ eyes.EyeConfig) error {
	atomic.StoreInt32(&f.unveilStarted, 1)
	f.startedAt.Store(time.Now().UnixNano())

	select {
	case f.unveilStart <- struct{}{}:
	default:
	}

	defer atomic.StoreInt32(&f.unveilDone, 1)

	if f.unveilFunc != nil {
		return f.unveilFunc(ctx)
	}

	<-ctx.Done()
	return ctx.Err()
}

func (f *fakeEye) Pause(_ context.Context) error { return nil }

func (f *fakeEye) Close(ctx context.Context) error {
	atomic.StoreInt32(&f.closeDone, 1)
	if f.closeFunc != nil {
		return f.closeFunc(ctx)
	}
	return nil
}

func (f *fakeEye) Observe() eyes.Metrics {
	return eyes.Metrics{EyeName: f.name, IsActive: atomic.LoadInt32(&f.unveilStarted) == 1}
}

// closeCalled reports whether Close ran at least once.
func (f *fakeEye) closeCalled() bool { return atomic.LoadInt32(&f.closeDone) == 1 }

// stubConfig satisfies eyes.EyeConfig for fake-eye tests.
type stubEyeConfig struct{}

func (stubEyeConfig) Validate() error { return nil }

// ---- fake-eye registry shim --------------------------------------------

var (
	fakeRegistryMu sync.Mutex
	fakeRegistry   = make(map[string]*fakeEye)
)

// registerFakeEye installs `fake` under `name` in the global eye registry.
// Each name may be registered only once per process; tests must generate
// unique names via uniqueEyeName.
func registerFakeEye(name string, fake *fakeEye) {
	fakeRegistryMu.Lock()
	defer fakeRegistryMu.Unlock()

	fakeRegistry[name] = fake

	if eyes.Exists(name) {
		return
	}

	eyes.Register(name, func(_ eyes.Dependencies) eyes.Eye {
		fakeRegistryMu.Lock()
		defer fakeRegistryMu.Unlock()
		return fakeRegistry[name]
	})
}

var eyeNameCounter atomic.Uint64

func uniqueEyeName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, eyeNameCounter.Add(1))
}

// ---- target resolver fake ----------------------------------------------

type fakeMultiResolver struct {
	byName map[string]*k8s.ResolvedTarget
}

func (f *fakeMultiResolver) Resolve(_ context.Context, target eyes.Target) (*k8s.ResolvedTarget, error) {
	if resolved, ok := f.byName[target.Name]; ok {
		return resolved, nil
	}
	return nil, fmt.Errorf("fake resolver: unknown target %q", target.Name)
}

func podWithUID(name, namespace, uid string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID(uid),
		},
	}
}

func resolvedFor(deploymentName, namespace string, pods ...corev1.Pod) *k8s.ResolvedTarget {
	return &k8s.ResolvedTarget{
		ResourceFound: true,
		ResourceKind:  "deployment",
		ResourceName:  deploymentName,
		Selector:      "app=" + deploymentName,
		Pods:          pods,
	}
}

func fakeTarget(deploymentName string) eyes.Target {
	return eyes.Target{Kind: "deployment", Name: deploymentName, Namespace: "default"}
}

func newMultiTestOrchestrator(t *testing.T, resolver k8s.TargetResolver) *Orchestrator {
	t.Helper()

	manager := &mockPodManager{}
	return newTestOrchestrator(t, manager, nil).
		withResolver(resolver)
}

// withResolver swaps in a resolver post-construction so we can reuse the
// single-eye orchestrator helpers.
func (o *Orchestrator) withResolver(resolver k8s.TargetResolver) *Orchestrator {
	o.resolver = resolver
	return o
}

// ---- tests -------------------------------------------------------------

func TestRunMulti_ContinueAggregatesErrors(t *testing.T) {
	nameA := uniqueEyeName("fake-continue-a")
	nameB := uniqueEyeName("fake-continue-b")

	errA := errors.New("eye A failed")
	errB := errors.New("eye B failed")

	eyeA := newFakeEye(nameA)
	eyeA.unveilFunc = func(_ context.Context) error { return errA }
	eyeB := newFakeEye(nameB)
	eyeB.unveilFunc = func(_ context.Context) error { return errB }

	registerFakeEye(nameA, eyeA)
	registerFakeEye(nameB, eyeB)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
			"api-b": resolvedFor("api-b", "default", podWithUID("pb", "default", "uid-b")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName: "continue-test",
		Duration:       2 * time.Second,
		FailurePolicy:  FailurePolicyContinue,
		BlastRadius:    50,
		MinHealthy:     0,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
			{Name: nameB, Target: fakeTarget("api-b"), Config: stubEyeConfig{}},
		},
	}

	err := orch.RunMulti(context.Background(), cfg)
	if err == nil {
		t.Fatal("RunMulti should surface aggregated errors")
	}

	if !errors.Is(err, errA) {
		t.Errorf("aggregated error should wrap errA: %v", err)
	}

	if !errors.Is(err, errB) {
		t.Errorf("aggregated error should wrap errB: %v", err)
	}

	if !eyeA.closeCalled() || !eyeB.closeCalled() {
		t.Error("both eyes should have been closed")
	}
}

func TestRunMulti_FailFastCancelsSiblings(t *testing.T) {
	nameA := uniqueEyeName("fake-ff-a")
	nameB := uniqueEyeName("fake-ff-b")

	errA := errors.New("eye A failed fast")

	eyeA := newFakeEye(nameA)
	eyeA.unveilFunc = func(_ context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return errA
	}

	bCancelled := make(chan struct{})
	eyeB := newFakeEye(nameB)
	eyeB.unveilFunc = func(ctx context.Context) error {
		<-ctx.Done()
		close(bCancelled)
		return ctx.Err()
	}

	registerFakeEye(nameA, eyeA)
	registerFakeEye(nameB, eyeB)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
			"api-b": resolvedFor("api-b", "default", podWithUID("pb", "default", "uid-b")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName: "failfast-test",
		Duration:       5 * time.Second,
		FailurePolicy:  FailurePolicyFailFast,
		BlastRadius:    50,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
			{Name: nameB, Target: fakeTarget("api-b"), Config: stubEyeConfig{}},
		},
	}

	start := time.Now()
	err := orch.RunMulti(context.Background(), cfg)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("RunMulti should surface fail-fast error")
	}

	if !errors.Is(err, errA) {
		t.Errorf("fail-fast error should wrap errA: %v", err)
	}

	if elapsed > 2*time.Second {
		t.Errorf("fail-fast took %v — should cancel siblings well before wall-clock", elapsed)
	}

	select {
	case <-bCancelled:
	case <-time.After(time.Second):
		t.Fatal("sibling B was not cancelled within 1s of A's failure")
	}

	if !eyeA.closeCalled() || !eyeB.closeCalled() {
		t.Error("both eyes must be closed after fail-fast")
	}
}

func TestRunMulti_StrictIsolationRejectsOverlap(t *testing.T) {
	nameA := uniqueEyeName("fake-strict-a")
	nameB := uniqueEyeName("fake-strict-b")

	eyeA := newFakeEye(nameA)
	eyeB := newFakeEye(nameB)

	registerFakeEye(nameA, eyeA)
	registerFakeEye(nameB, eyeB)

	sharedPod := podWithUID("shared", "default", "uid-shared")

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", sharedPod),
			"api-b": resolvedFor("api-b", "default", sharedPod),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName:  "strict-isolation",
		Duration:        2 * time.Second,
		FailurePolicy:   FailurePolicyContinue,
		StrictIsolation: true,
		BlastRadius:     100,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
			{Name: nameB, Target: fakeTarget("api-b"), Config: stubEyeConfig{}},
		},
	}

	err := orch.RunMulti(context.Background(), cfg)
	if err == nil {
		t.Fatal("RunMulti should reject overlapping targets under strict isolation")
	}

	if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
		t.Errorf("strict-isolation error should wrap ErrInvalidConfiguration, got %v", err)
	}

	if atomic.LoadInt32(&eyeA.unveilStarted) == 1 || atomic.LoadInt32(&eyeB.unveilStarted) == 1 {
		t.Error("no eye should have started when strict isolation rejects")
	}
}

func TestRunMulti_StaggerDelaysLaunch(t *testing.T) {
	nameA := uniqueEyeName("fake-stagger-a")
	nameB := uniqueEyeName("fake-stagger-b")

	eyeA := newFakeEye(nameA)
	eyeB := newFakeEye(nameB)

	registerFakeEye(nameA, eyeA)
	registerFakeEye(nameB, eyeB)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
			"api-b": resolvedFor("api-b", "default", podWithUID("pb", "default", "uid-b")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName:  "stagger-test",
		Duration:        300 * time.Millisecond,
		FailurePolicy:   FailurePolicyContinue,
		StaggerInterval: 100 * time.Millisecond,
		BlastRadius:     100,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
			{Name: nameB, Target: fakeTarget("api-b"), Config: stubEyeConfig{}},
		},
	}

	_ = orch.RunMulti(context.Background(), cfg)

	startA := eyeA.startedAt.Load()
	startB := eyeB.startedAt.Load()

	if startA == 0 || startB == 0 {
		t.Fatalf("both eyes should have started: startA=%d startB=%d", startA, startB)
	}

	gap := time.Duration(startB - startA)
	if gap < 80*time.Millisecond {
		t.Errorf("stagger gap = %v, want ≥ 80ms", gap)
	}
}

func TestRunMulti_RootContextCancelStopsInFlight(t *testing.T) {
	nameA := uniqueEyeName("fake-ctxcancel-a")

	eyeA := newFakeEye(nameA)
	registerFakeEye(nameA, eyeA)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := MultiConfig{
		ExperimentName: "ctx-cancel",
		Duration:       10 * time.Second,
		FailurePolicy:  FailurePolicyContinue,
		BlastRadius:    100,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
		},
	}

	done := make(chan error, 1)
	go func() { done <- orch.RunMulti(ctx, cfg) }()

	select {
	case <-eyeA.unveilStart:
	case <-time.After(time.Second):
		t.Fatal("eye A never started")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunMulti did not return within 2s of root ctx cancel")
	}

	if !eyeA.closeCalled() {
		t.Error("Close must run even after root ctx cancel")
	}
}

func TestRunMulti_PanicRecoveredAndClosed(t *testing.T) {
	nameA := uniqueEyeName("fake-panic-a")

	eyeA := newFakeEye(nameA)
	eyeA.unveilFunc = func(_ context.Context) error {
		panic("oops — Viy's gaze met a void")
	}
	registerFakeEye(nameA, eyeA)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName: "panic-test",
		Duration:       time.Second,
		FailurePolicy:  FailurePolicyContinue,
		BlastRadius:    100,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
		},
	}

	err := orch.RunMulti(context.Background(), cfg)
	if err == nil {
		t.Fatal("RunMulti should surface recovered panic as error")
	}

	if !eyeA.closeCalled() {
		t.Error("Close must run even after panic")
	}
}

func TestRunMulti_PerEyeDurationRespected(t *testing.T) {
	nameA := uniqueEyeName("fake-perdur-a")

	eyeA := newFakeEye(nameA)
	registerFakeEye(nameA, eyeA)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName: "per-eye-duration",
		Duration:       5 * time.Second,
		FailurePolicy:  FailurePolicyContinue,
		BlastRadius:    100,
		Eyes: []EyeRunSpec{
			{
				Name:     nameA,
				Target:   fakeTarget("api-a"),
				Config:   stubEyeConfig{},
				Duration: 150 * time.Millisecond,
			},
		},
	}

	start := time.Now()
	_ = orch.RunMulti(context.Background(), cfg)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Errorf("elapsed = %v, want close to 150ms (per-eye duration should shorten wall-clock)", elapsed)
	}
}

func TestRunMulti_NoEyesErrors(t *testing.T) {
	resolver := &fakeMultiResolver{}
	orch := newMultiTestOrchestrator(t, resolver)

	err := orch.RunMulti(context.Background(), MultiConfig{
		ExperimentName: "empty",
		Duration:       time.Second,
	})

	if err == nil {
		t.Fatal("RunMulti should error when no eyes are configured")
	}
}

func TestRunMulti_DryRunDoesNotExecute(t *testing.T) {
	nameA := uniqueEyeName("fake-dream-a")

	eyeA := newFakeEye(nameA)
	registerFakeEye(nameA, eyeA)

	resolver := &fakeMultiResolver{
		byName: map[string]*k8s.ResolvedTarget{
			"api-a": resolvedFor("api-a", "default", podWithUID("pa", "default", "uid-a")),
		},
	}

	orch := newMultiTestOrchestrator(t, resolver)

	cfg := MultiConfig{
		ExperimentName: "dream",
		Duration:       time.Second,
		FailurePolicy:  FailurePolicyContinue,
		BlastRadius:    100,
		DryRun:         true,
		Eyes: []EyeRunSpec{
			{Name: nameA, Target: fakeTarget("api-a"), Config: stubEyeConfig{}},
		},
	}

	if err := orch.RunMulti(context.Background(), cfg); err != nil {
		t.Fatalf("RunMulti(dry-run) error = %v", err)
	}

	if atomic.LoadInt32(&eyeA.unveilStarted) == 1 {
		t.Error("dry-run must not invoke Unveil")
	}

	if eyeA.closeCalled() {
		t.Error("dry-run must not invoke Close")
	}
}
