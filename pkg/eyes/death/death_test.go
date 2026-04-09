package death

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
		CPUStressPercent: 80,
		Duration:         2 * time.Minute,
		Workers:          4,
	}
	invalid := &Config{}

	eyestest.RunContractTests(t, NewDeathEye, valid, invalid)
}

// --- Config validation ---

func TestConfig_Validate_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "cpu only",
			config: Config{
				CPUStressPercent: 50,
				Duration:         time.Minute,
				Workers:          1,
			},
		},
		{
			name: "memory only",
			config: Config{
				MemoryStressPercent: 70,
				Duration:            time.Minute,
				Workers:             2,
			},
		},
		{
			name: "disk io only",
			config: Config{
				DiskIOBytes: 1024 * 1024,
				Duration:    time.Minute,
				Workers:     1,
			},
		},
		{
			name: "all stressors with ramp up",
			config: Config{
				CPUStressPercent:    80,
				MemoryStressPercent: 70,
				DiskIOBytes:         512 * 1024,
				Duration:            2 * time.Minute,
				RampUp:              30 * time.Second,
				Workers:             4,
			},
		},
		{
			name: "zero ramp up",
			config: Config{
				CPUStressPercent: 100,
				Duration:         time.Minute,
				Workers:          1,
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
			name:    "no stressors enabled",
			config:  Config{Duration: time.Minute, Workers: 1},
			wantErr: "at least one stress type",
		},
		{
			name:    "cpu below minimum",
			config:  Config{CPUStressPercent: -1, Duration: time.Minute, Workers: 1},
			wantErr: "cpuStress must be between",
		},
		{
			name:    "cpu above maximum",
			config:  Config{CPUStressPercent: 101, Duration: time.Minute, Workers: 1},
			wantErr: "cpuStress must be between",
		},
		{
			name:    "memory above maximum",
			config:  Config{MemoryStressPercent: 200, Duration: time.Minute, Workers: 1},
			wantErr: "memoryStress must be between",
		},
		{
			name:    "negative disk io",
			config:  Config{DiskIOBytes: -1, CPUStressPercent: 50, Duration: time.Minute, Workers: 1},
			wantErr: "diskIOBytes must be non-negative",
		},
		{
			name:    "zero duration",
			config:  Config{CPUStressPercent: 50, Workers: 1},
			wantErr: "duration must be positive",
		},
		{
			name:    "negative ramp up",
			config:  Config{CPUStressPercent: 50, Duration: time.Minute, RampUp: -time.Second, Workers: 1},
			wantErr: "rampUp must be non-negative",
		},
		{
			name:    "ramp up exceeds duration",
			config:  Config{CPUStressPercent: 50, Duration: time.Minute, RampUp: 2 * time.Minute, Workers: 1},
			wantErr: "rampUp must be less than duration",
		},
		{
			name:    "ramp up equals duration",
			config:  Config{CPUStressPercent: 50, Duration: time.Minute, RampUp: time.Minute, Workers: 1},
			wantErr: "rampUp must be less than duration",
		},
		{
			name:    "zero workers",
			config:  Config{CPUStressPercent: 50, Duration: time.Minute, Workers: 0},
			wantErr: "workers must be at least",
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

// --- Stress command building ---

func TestBuildStressCommand(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected []string
	}{
		{
			name: "cpu only",
			config: Config{
				CPUStressPercent: 80,
				Duration:         2 * time.Minute,
				Workers:          4,
			},
			expected: []string{
				"stress-ng",
				"--cpu", "4", "--cpu-load", "80",
				"--timeout", "120",
			},
		},
		{
			name: "memory only",
			config: Config{
				MemoryStressPercent: 70,
				Duration:            time.Minute,
				Workers:             2,
			},
			expected: []string{
				"stress-ng",
				"--vm", "2", "--vm-bytes", "70%",
				"--timeout", "60",
			},
		},
		{
			name: "disk io only",
			config: Config{
				DiskIOBytes: 1048576,
				Duration:    time.Minute,
				Workers:     3,
			},
			expected: []string{
				"stress-ng",
				"--hdd", "3", "--hdd-bytes", "1048576",
				"--timeout", "60",
			},
		},
		{
			name: "all stressors with ramp up",
			config: Config{
				CPUStressPercent:    80,
				MemoryStressPercent: 70,
				DiskIOBytes:         1048576,
				Duration:            2 * time.Minute,
				RampUp:              30 * time.Second,
				Workers:             4,
			},
			expected: []string{
				"stress-ng",
				"--cpu", "4", "--cpu-load", "80",
				"--vm", "4", "--vm-bytes", "70%",
				"--hdd", "4", "--hdd-bytes", "1048576",
				"--ramp-time", "30",
				"--timeout", "120",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildStressCommand(&tt.config)
			assertSliceEqual(t, tt.expected, result)
		})
	}
}

// --- Ephemeral container building ---

func TestBuildEphemeralContainer(t *testing.T) {
	command := []string{"stress-ng", "--cpu", "4", "--timeout", "60"}
	container := buildEphemeralContainer("viy-death-test", command)

	if container.Name != "viy-death-test" {
		t.Errorf("Name = %q, want %q", container.Name, "viy-death-test")
	}

	if container.Image != stressImage {
		t.Errorf("Image = %q, want %q", container.Image, stressImage)
	}

	assertSliceEqual(t, command, container.Command)
}

// --- Unveil behavior ---

func TestUnveil_InjectsStressIntoAllPods(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "api-server-abc123"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "api-server-def456"}},
	}

	podMgr := &fakePodManager{pods: pods}
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(podMgr, ephMgr)

	cfg := &Config{
		CPUStressPercent: 80,
		Duration:         time.Minute,
		Workers:          2,
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
		CPUStressPercent: 80,
		Duration:         time.Minute,
		Workers:          1,
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
		CPUStressPercent: 50,
		Duration:         time.Minute,
		Workers:          1,
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
		failForPods: map[string]bool{"pod-failure": true},
	}
	eye := newTestEye(&fakePodManager{pods: pods}, ephMgr)

	cfg := &Config{
		CPUStressPercent: 80,
		Duration:         time.Minute,
		Workers:          1,
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
		if contains(truth, "partial revelation") {
			foundPartialTruth = true
			break
		}
	}

	if !foundPartialTruth {
		t.Errorf("expected partial failure truth, got truths: %v", metrics.TruthsRevealed)
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
		CPUStressPercent: 50,
		Duration:         time.Minute,
		Workers:          1,
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

func TestPause_TerminatesStress(t *testing.T) {
	ephMgr := &fakeEphemeralContainerManager{}
	eye := newTestEye(&fakePodManager{}, ephMgr)

	eye.mu.Lock()
	eye.injected = []injectedStress{
		{namespace: "default", podName: "pod-1", containerName: "viy-death-pod-1"},
		{namespace: "default", podName: "pod-2", containerName: "viy-death-pod-2"},
	}
	eye.mu.Unlock()

	err := eye.Pause(context.Background())
	if err != nil {
		t.Fatalf("Pause() returned error: %v", err)
	}

	if ephMgr.execCount != 2 {
		t.Errorf("expected 2 exec calls to kill stress, got %d", ephMgr.execCount)
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

func newTestEye(podMgr eyes.PodManager, ephMgr eyes.EphemeralContainerManager) *DeathEye {
	logger, _ := zap.NewDevelopment()
	return NewDeathEye(eyes.Dependencies{
		PodManager:                podMgr,
		EphemeralContainerManager: ephMgr,
		Logger:                    logger,
	}).(*DeathEye)
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
	addCount    int
	execCount   int
	failForPods map[string]bool
	cancelAfter int
	cancelFunc  context.CancelFunc
}

func (f *fakeEphemeralContainerManager) AddEphemeralContainer(_ context.Context, _, podName string, _ corev1.EphemeralContainer) error {
	if f.failForPods[podName] {
		return fmt.Errorf("injection failed for pod %s", podName)
	}

	f.addCount++

	if f.cancelFunc != nil && f.addCount >= f.cancelAfter {
		f.cancelFunc()
	}

	return nil
}

func (f *fakeEphemeralContainerManager) ExecInContainer(_ context.Context, _, _, _ string, _ []string) error {
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
