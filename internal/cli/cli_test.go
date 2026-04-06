package cli

import (
	"testing"
	"time"

	"github.com/oragazz0/viy/internal/eyes/disintegration"
	"github.com/oragazz0/viy/internal/state"
)

func TestParsePercentage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "with percent sign", input: "30%", want: 30},
		{name: "without percent sign", input: "50", want: 50},
		{name: "100 percent", input: "100%", want: 100},
		{name: "1 percent", input: "1%", want: 1},
		{name: "zero", input: "0%", wantErr: true},
		{name: "over 100", input: "150%", wantErr: true},
		{name: "not a number", input: "abc", wantErr: true},
		{name: "negative", input: "-5%", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePercentage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePercentage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePercentage(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSelectorFromTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "deployment target", target: "deployment/nginx", want: "app=nginx"},
		{name: "pod target", target: "pod/api-abc", want: "app=api-abc"},
		{name: "bare name", target: "nginx", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSelectorFromTarget(tt.target)
			if got != tt.want {
				t.Errorf("parseSelectorFromTarget(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestParseKindFromTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "deployment", target: "deployment/nginx", want: "deployment"},
		{name: "statefulset", target: "statefulset/db", want: "statefulset"},
		{name: "bare name", target: "nginx", want: "pod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseKindFromTarget(tt.target)
			if got != tt.want {
				t.Errorf("parseKindFromTarget(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestParseNameFromTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{name: "with kind", target: "deployment/nginx", want: "nginx"},
		{name: "bare name", target: "nginx", want: "nginx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNameFromTarget(tt.target)
			if got != tt.want {
				t.Errorf("parseNameFromTarget(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestBuildDisintegrationConfig_Default(t *testing.T) {
	config := buildDisintegrationConfig("")

	if config.PodKillCount != 1 {
		t.Errorf("PodKillCount = %d, want 1", config.PodKillCount)
	}

	if config.Strategy != "random" {
		t.Errorf("Strategy = %q, want %q", config.Strategy, "random")
	}
}

func TestBuildDisintegrationConfig_CustomValues(t *testing.T) {
	config := buildDisintegrationConfig("podKillCount=3,strategy=sequential,interval=30s,gracePeriod=5s")

	if config.PodKillCount != 3 {
		t.Errorf("PodKillCount = %d, want 3", config.PodKillCount)
	}

	if config.Strategy != "sequential" {
		t.Errorf("Strategy = %q, want %q", config.Strategy, "sequential")
	}

	if config.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", config.Interval)
	}

	if config.GracePeriod != 5*time.Second {
		t.Errorf("GracePeriod = %v, want 5s", config.GracePeriod)
	}
}

func TestBuildDisintegrationConfig_Percentage(t *testing.T) {
	config := buildDisintegrationConfig("podKillPercentage=30%")

	if config.PodKillPercentage != 30 {
		t.Errorf("PodKillPercentage = %d, want 30", config.PodKillPercentage)
	}

	if config.PodKillCount != 0 {
		t.Errorf("PodKillCount = %d, want 0 (cleared by percentage)", config.PodKillCount)
	}
}

func TestBuildDisintegrationConfig_InvalidValues(t *testing.T) {
	config := buildDisintegrationConfig("podKillCount=abc,strategy=invalid,interval=bad")

	expected := disintegration.Config{
		PodKillCount: 1,
		Strategy:     "random",
	}

	if config.PodKillCount != expected.PodKillCount {
		t.Errorf("PodKillCount = %d, want %d (default)", config.PodKillCount, expected.PodKillCount)
	}

	if config.Strategy != expected.Strategy {
		t.Errorf("Strategy = %q, want %q (default)", config.Strategy, expected.Strategy)
	}
}

func TestBuildDisintegrationConfig_MalformedPairs(t *testing.T) {
	config := buildDisintegrationConfig("noequalssign,=nokey,podKillCount=2")

	if config.PodKillCount != 2 {
		t.Errorf("PodKillCount = %d, want 2 (valid pair should still parse)", config.PodKillCount)
	}
}

func TestRunVision_NoExperiments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runVision(false)
	if err != nil {
		t.Fatalf("runVision() error = %v", err)
	}
}

func TestRunVision_WithExperiments(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	store, err := state.NewStore()
	if err != nil {
		t.Fatal(err)
	}

	experiments := []state.Experiment{
		{
			ID:        "exp-1",
			Status:    state.StatusUnveiling,
			Eyes:      []string{"disintegration"},
			Target:    "api",
			Namespace: "default",
			StartTime: time.Now(),
			Duration:  5 * time.Minute,
		},
		{
			ID:        "exp-2",
			Status:    state.StatusRevealed,
			Eyes:      []string{"disintegration"},
			Target:    "web",
			Namespace: "staging",
			StartTime: time.Now(),
			Duration:  10 * time.Minute,
		},
	}

	if err := store.Save(experiments); err != nil {
		t.Fatal(err)
	}

	if err := runVision(false); err != nil {
		t.Fatalf("runVision(false) error = %v", err)
	}

	if err := runVision(true); err != nil {
		t.Fatalf("runVision(true) error = %v", err)
	}
}

func TestRunSlumber_NoExperiments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := runSlumber(true, "", false)
	if err != nil {
		t.Fatalf("runSlumber() error = %v", err)
	}
}

func TestRunSlumber_StopsExperiment(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	store, err := state.NewStore()
	if err != nil {
		t.Fatal(err)
	}

	experiments := []state.Experiment{
		{
			ID:        "exp-1",
			Status:    state.StatusUnveiling,
			Eyes:      []string{"disintegration"},
			Target:    "api",
			Namespace: "default",
			StartTime: time.Now(),
		},
	}

	if err := store.Save(experiments); err != nil {
		t.Fatal(err)
	}

	if err := runSlumber(false, "exp-1", false); err != nil {
		t.Fatalf("runSlumber() error = %v", err)
	}

	loaded, _ := store.Load()
	if loaded[0].Status != state.StatusRevealed {
		t.Errorf("experiment status = %q, want %q", loaded[0].Status, state.StatusRevealed)
	}
}

func TestRunSlumber_AllExperiments(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	store, err := state.NewStore()
	if err != nil {
		t.Fatal(err)
	}

	experiments := []state.Experiment{
		{ID: "exp-1", Status: state.StatusUnveiling, StartTime: time.Now()},
		{ID: "exp-2", Status: state.StatusUnveiling, StartTime: time.Now()},
	}

	if err := store.Save(experiments); err != nil {
		t.Fatal(err)
	}

	if err := runSlumber(true, "", false); err != nil {
		t.Fatalf("runSlumber(all) error = %v", err)
	}

	loaded, _ := store.Load()
	for _, exp := range loaded {
		if exp.Status != state.StatusRevealed {
			t.Errorf("experiment %s status = %q, want %q", exp.ID, exp.Status, state.StatusRevealed)
		}
	}
}

func TestRunUnveil_ProtectedNamespace(t *testing.T) {
	err := runUnveil("disintegration", "deployment/api", "kube-system", time.Minute, "30%", "", false, 1)
	if err == nil {
		t.Fatal("runUnveil() should fail for protected namespace")
	}
}

func TestNewRootCommand_HasSubcommands(t *testing.T) {
	root := newRootCommand()

	expected := []string{"unveil", "dream", "slumber", "vision", "version"}

	for _, name := range expected {
		found := false
		for _, cmd := range root.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("root command missing subcommand %q", name)
		}
	}
}

func TestProtectedNamespaces(t *testing.T) {
	namespaces := []string{"kube-system", "kube-public", "kube-node-lease"}

	for _, ns := range namespaces {
		if !protectedNamespaces[ns] {
			t.Errorf("namespace %q should be protected", ns)
		}
	}

	if protectedNamespaces["default"] {
		t.Error("default namespace should not be protected")
	}
}
