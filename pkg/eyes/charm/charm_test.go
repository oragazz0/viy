package charm

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
	"github.com/oragazz0/viy/pkg/eyes"
	"github.com/oragazz0/viy/pkg/eyes/eyestest"
)

func TestContract(t *testing.T) {
	valid := &Config{
		Latency:  500 * time.Millisecond,
		Duration: 2 * time.Minute,
	}
	invalid := &Config{}

	eyestest.RunContractTests(t, NewCharmEye, valid, invalid)
}

// --- Config validation ---

func TestConfig_Validate_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "latency only",
			config: Config{
				Latency:  200 * time.Millisecond,
				Duration: time.Minute,
			},
		},
		{
			name: "packet loss only",
			config: Config{
				PacketLoss: 10.0,
				Duration:   time.Minute,
			},
		},
		{
			name: "corruption only",
			config: Config{
				Corruption: 5.0,
				Duration:   time.Minute,
			},
		},
		{
			name: "latency with jitter",
			config: Config{
				Latency:  500 * time.Millisecond,
				Jitter:   100 * time.Millisecond,
				Duration: time.Minute,
			},
		},
		{
			name: "all parameters",
			config: Config{
				Latency:    500 * time.Millisecond,
				Jitter:     100 * time.Millisecond,
				PacketLoss: 10.0,
				Corruption: 2.0,
				Duration:   2 * time.Minute,
			},
		},
		{
			name: "explicit interface",
			config: Config{
				Latency:   200 * time.Millisecond,
				Duration:  time.Minute,
				Interface: "ens5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
		})
	}
}

func TestConfig_Validate_InvalidConfigs(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "no chaos parameters",
			config:  Config{Duration: time.Minute},
			wantErr: "at least one chaos parameter",
		},
		{
			name:    "negative latency",
			config:  Config{Latency: -time.Millisecond, Duration: time.Minute},
			wantErr: "latency must be non-negative",
		},
		{
			name:    "jitter without latency",
			config:  Config{Jitter: 100 * time.Millisecond, PacketLoss: 5.0, Duration: time.Minute},
			wantErr: "jitter requires latency",
		},
		{
			name:    "negative jitter",
			config:  Config{Latency: 200 * time.Millisecond, Jitter: -time.Millisecond, Duration: time.Minute},
			wantErr: "jitter must be non-negative",
		},
		{
			name:    "packet loss below zero",
			config:  Config{PacketLoss: -1.0, Duration: time.Minute},
			wantErr: "packetLoss must be between",
		},
		{
			name:    "packet loss above 100",
			config:  Config{PacketLoss: 101.0, Duration: time.Minute},
			wantErr: "packetLoss must be between",
		},
		{
			name:    "corruption above 100",
			config:  Config{Corruption: 150.0, Duration: time.Minute},
			wantErr: "corruption must be between",
		},
		{
			name:    "zero duration",
			config:  Config{Latency: 100 * time.Millisecond},
			wantErr: "duration must be positive",
		},
		{
			name:    "negative duration",
			config:  Config{Latency: 100 * time.Millisecond, Duration: -time.Second},
			wantErr: "duration must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
				t.Errorf("expected ErrInvalidConfiguration, got: %v", err)
			}

			if tt.wantErr != "" && !contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// --- tc command building ---

func TestBuildApplyCommand_ExplicitInterface(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected []string
	}{
		{
			name: "latency only",
			config: Config{
				Latency:   500 * time.Millisecond,
				Interface: "eth0",
			},
			expected: []string{
				"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
				"delay", "500ms",
			},
		},
		{
			name: "latency with jitter",
			config: Config{
				Latency:   500 * time.Millisecond,
				Jitter:    100 * time.Millisecond,
				Interface: "eth0",
			},
			expected: []string{
				"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
				"delay", "500ms", "100ms",
			},
		},
		{
			name: "packet loss only",
			config: Config{
				PacketLoss: 10.0,
				Interface:  "eth0",
			},
			expected: []string{
				"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
				"loss", "10.00%",
			},
		},
		{
			name: "corruption only",
			config: Config{
				Corruption: 2.5,
				Interface:  "eth0",
			},
			expected: []string{
				"tc", "qdisc", "add", "dev", "eth0", "root", "netem",
				"corrupt", "2.50%",
			},
		},
		{
			name: "all parameters",
			config: Config{
				Latency:    500 * time.Millisecond,
				Jitter:     100 * time.Millisecond,
				PacketLoss: 10.0,
				Corruption: 2.0,
				Interface:  "ens5",
			},
			expected: []string{
				"tc", "qdisc", "add", "dev", "ens5", "root", "netem",
				"delay", "500ms", "100ms",
				"loss", "10.00%",
				"corrupt", "2.00%",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildApplyCommand(&tt.config, tt.config.Interface)
			assertSliceEqual(t, tt.expected, result)
		})
	}
}

func TestBuildApplyCommand_AutoDetect(t *testing.T) {
	cfg := &Config{
		Latency:    500 * time.Millisecond,
		PacketLoss: 10.0,
	}

	result := buildApplyCommand(cfg, "")

	if len(result) != 3 {
		t.Fatalf("expected shell wrapper with 3 args, got %d: %v", len(result), result)
	}

	if result[0] != "sh" || result[1] != "-c" {
		t.Errorf("expected sh -c wrapper, got: %v", result[:2])
	}

	script := result[2]
	if !contains(script, "ip -o route show default") {
		t.Error("auto-detect script should use ip route")
	}

	if !contains(script, "tc qdisc add") {
		t.Error("script should contain tc qdisc add")
	}

	if !contains(script, "delay 500ms") {
		t.Errorf("script should contain delay 500ms, got: %s", script)
	}

	if !contains(script, "loss 10.00%") {
		t.Errorf("script should contain loss 10.00%%, got: %s", script)
	}
}

func TestBuildCleanupCommand_ExplicitInterface(t *testing.T) {
	expected := []string{"tc", "qdisc", "del", "dev", "eth0", "root"}
	result := buildCleanupCommand("eth0")
	assertSliceEqual(t, expected, result)
}

func TestBuildCleanupCommand_AutoDetect(t *testing.T) {
	result := buildCleanupCommand("")

	if len(result) != 3 || result[0] != "sh" {
		t.Fatalf("expected shell wrapper, got: %v", result)
	}

	script := result[2]
	if !contains(script, "tc qdisc del") {
		t.Error("cleanup script should contain tc qdisc del")
	}
}

// --- Netshoot container ---

func TestBuildNetshootContainer(t *testing.T) {
	container := buildNetshootContainer("viy-charm-test")

	if container.Name != "viy-charm-test" {
		t.Errorf("Name = %q, want %q", container.Name, "viy-charm-test")
	}

	if container.Image != netshootImage {
		t.Errorf("Image = %q, want %q", container.Image, netshootImage)
	}

	assertSliceEqual(t, []string{"sleep", "infinity"}, container.Command)

	if container.SecurityContext == nil {
		t.Fatal("SecurityContext must be set")
	}

	if container.SecurityContext.Capabilities == nil {
		t.Fatal("Capabilities must be set")
	}

	caps := container.SecurityContext.Capabilities.Add
	if len(caps) != 1 || caps[0] != "NET_ADMIN" {
		t.Errorf("expected [NET_ADMIN], got %v", caps)
	}
}

// --- Unveil behavior ---

func TestUnveil_AppliesCharmToAllPods(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "api-server-abc123"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-server-def456"}},
	}

	podMgr := &fakePodManager{pods: pods}
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(podMgr, ephMgr)

	cfg := &Config{
		Latency:   500 * time.Millisecond,
		Duration:  time.Minute,
		Interface: "eth0",
	}

	target := eyes.Target{
		Namespace: "default",
		Selector:  "app=api",
	}

	err := eye.Unveil(context.Background(), target, cfg)
	if err != nil {
		t.Fatalf("Unveil() returned error: %v", err)
	}

	if ephMgr.addCount != 2 {
		t.Errorf("expected 2 ephemeral containers injected, got %d", ephMgr.addCount)
	}

	if ephMgr.execCount != 2 {
		t.Errorf("expected 2 tc exec calls, got %d", ephMgr.execCount)
	}

	metrics := eye.Observe()
	if metrics.TargetsAffected != 2 {
		t.Errorf("TargetsAffected = %d, want 2", metrics.TargetsAffected)
	}

	if !metrics.IsActive {
		t.Error("eye should be active after Unveil")
	}
}

func TestUnveil_NoPods_ReturnsError(t *testing.T) {
	podMgr := &fakePodManager{pods: nil}
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(podMgr, ephMgr)

	cfg := &Config{
		Latency:   200 * time.Millisecond,
		Duration:  time.Minute,
		Interface: "eth0",
	}

	target := eyes.Target{
		Namespace: "default",
		Selector:  "app=missing",
	}

	err := eye.Unveil(context.Background(), target, cfg)
	if !errors.Is(err, viyerrors.ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound, got: %v", err)
	}
}

func TestUnveil_GetPodsFails_ReturnsError(t *testing.T) {
	podMgr := &fakePodManager{err: errors.New("connection refused")}
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(podMgr, ephMgr)

	cfg := &Config{
		Latency:   200 * time.Millisecond,
		Duration:  time.Minute,
		Interface: "eth0",
	}

	err := eye.Unveil(context.Background(), eyes.Target{Namespace: "default", Selector: "app=test"}, cfg)
	if err == nil {
		t.Fatal("expected error when GetPods fails")
	}
}

func TestUnveil_PartialFailure_SurfacesTruth(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-success"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-failure"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-success2"}},
	}

	ephMgr := &fakeEphemeralContainerManager{
		failAddForPods: map[string]bool{"pod-failure": true},
	}
	eye := newTestEye(&fakePodManager{pods: pods}, ephMgr)

	cfg := &Config{
		Latency:   500 * time.Millisecond,
		Duration:  time.Minute,
		Interface: "eth0",
	}

	err := eye.Unveil(context.Background(), eyes.Target{Namespace: "default", Selector: "app=test"}, cfg)
	if err != nil {
		t.Fatalf("Unveil should succeed on partial failure, got: %v", err)
	}

	metrics := eye.Observe()
	if metrics.TargetsAffected != 2 {
		t.Errorf("TargetsAffected = %d, want 2", metrics.TargetsAffected)
	}

	if metrics.ErrorsTotal != 1 {
		t.Errorf("ErrorsTotal = %d, want 1", metrics.ErrorsTotal)
	}

	foundPartialTruth := false
	for _, truth := range metrics.TruthsRevealed {
		if contains(truth, "partial enchantment") {
			foundPartialTruth = true
			break
		}
	}

	if !foundPartialTruth {
		t.Errorf("expected partial enchantment truth, got truths: %v", metrics.TruthsRevealed)
	}
}

func TestUnveil_ExecFails_CountsAsFailure(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}},
	}

	ephMgr := &fakeEphemeralContainerManager{
		failExecForPods: map[string]bool{"pod-1": true},
	}
	eye := newTestEye(&fakePodManager{pods: pods}, ephMgr)

	cfg := &Config{
		Latency:   200 * time.Millisecond,
		Duration:  time.Minute,
		Interface: "eth0",
	}

	err := eye.Unveil(context.Background(), eyes.Target{Namespace: "default", Selector: "app=test"}, cfg)
	if err != nil {
		t.Fatalf("Unveil should succeed on partial failure, got: %v", err)
	}

	metrics := eye.Observe()
	if metrics.TargetsAffected != 0 {
		t.Errorf("TargetsAffected = %d, want 0", metrics.TargetsAffected)
	}

	if metrics.ErrorsTotal != 1 {
		t.Errorf("ErrorsTotal = %d, want 1", metrics.ErrorsTotal)
	}
}

func TestUnveil_CancelledContext_StopsEarly(t *testing.T) {
	pods := make([]corev1.Pod, 10)
	for i := range pods {
		pods[i] = corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("pod-%d", i)}}
	}

	ctx, cancel := context.WithCancel(context.Background())
	ephMgr := &fakeEphemeralContainerManager{cancelAfter: 3, cancelFunc: cancel}
	eye := newTestEye(&fakePodManager{pods: pods}, ephMgr)

	cfg := &Config{
		Latency:   200 * time.Millisecond,
		Duration:  time.Minute,
		Interface: "eth0",
	}

	_ = eye.Unveil(ctx, eyes.Target{Namespace: "default", Selector: "app=test"}, cfg)

	metrics := eye.Observe()
	if metrics.TargetsAffected >= 10 {
		t.Errorf("expected early stop, but all %d pods were affected", metrics.TargetsAffected)
	}
}

// --- Close / Pause ---

func TestClose_SetsInactive(t *testing.T) {
	eye := newTestEye(&fakePodManager{}, &fakeEphemeralContainerManager{})
	eye.active.Store(true)

	err := eye.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if eye.Observe().IsActive {
		t.Error("eye should be inactive after Close")
	}
}

func TestClose_RemovesTcRules(t *testing.T) {
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(&fakePodManager{}, ephMgr)

	eye.mu.Lock()
	eye.charmed = []charmedPod{
		{namespace: "default", podName: "pod-1", containerName: "viy-charm-pod-1", iface: "eth0"},
		{namespace: "default", podName: "pod-2", containerName: "viy-charm-pod-2", iface: "eth0"},
	}
	eye.mu.Unlock()

	err := eye.Close(context.Background())
	if err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if ephMgr.execCount != 2 {
		t.Errorf("expected 2 cleanup exec calls, got %d", ephMgr.execCount)
	}
}

func TestPause_RemovesTcRules(t *testing.T) {
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(&fakePodManager{}, ephMgr)

	eye.mu.Lock()
	eye.charmed = []charmedPod{
		{namespace: "default", podName: "pod-1", containerName: "viy-charm-pod-1", iface: "eth0"},
	}
	eye.mu.Unlock()

	err := eye.Pause(context.Background())
	if err != nil {
		t.Fatalf("Pause() returned error: %v", err)
	}

	if ephMgr.execCount != 1 {
		t.Errorf("expected 1 cleanup exec call, got %d", ephMgr.execCount)
	}
}

// --- truncatePodName ---

func TestTruncatePodName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "short"},
		{"exactly8", "exactly8"},
		{"api-server-abc123def", "api-serv"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncatePodName(tt.input)
			if result != tt.expected {
				t.Errorf("truncatePodName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- helpers ---

func newTestEye(podMgr eyes.PodManager, ephMgr eyes.EphemeralContainerManager) *CharmEye {
	logger, _ := zap.NewDevelopment()
	return NewCharmEye(eyes.Dependencies{
		PodManager:                podMgr,
		EphemeralContainerManager: ephMgr,
		Logger:                    logger,
	}).(*CharmEye)
}

type fakePodManager struct {
	pods []corev1.Pod
	err  error
}

func (f *fakePodManager) GetPods(_ context.Context, _, _ string) ([]corev1.Pod, error) {
	return f.pods, f.err
}

func (f *fakePodManager) DeletePod(_ context.Context, _, _ string, _ int64) error {
	return nil
}

type fakeEphemeralContainerManager struct {
	addCount        int
	execCount       int
	failAddForPods  map[string]bool
	failExecForPods map[string]bool
	cancelAfter     int
	cancelFunc      context.CancelFunc
}

func (f *fakeEphemeralContainerManager) AddEphemeralContainer(_ context.Context, _, podName string, _ corev1.EphemeralContainer) error {
	if f.failAddForPods[podName] {
		return fmt.Errorf("injection failed for pod %s", podName)
	}

	f.addCount++

	if f.cancelFunc != nil && f.addCount >= f.cancelAfter {
		f.cancelFunc()
	}

	return nil
}

func (f *fakeEphemeralContainerManager) ExecInContainer(_ context.Context, _, podName, _ string, _ []string) error {
	if f.failExecForPods[podName] {
		return fmt.Errorf("exec failed for pod %s", podName)
	}

	f.execCount++
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func assertSliceEqual(t *testing.T, expected, actual []string) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Errorf("length mismatch: want %d, got %d\nwant: %v\ngot:  %v", len(expected), len(actual), expected, actual)
		return
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("index %d: want %q, got %q", i, expected[i], actual[i])
		}
	}
}
