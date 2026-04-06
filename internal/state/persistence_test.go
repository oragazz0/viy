package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStore_SaveAndLoad(t *testing.T) {
	store := tempStore(t)

	experiments := []Experiment{
		{
			ID:        "exp-abc",
			Status:    StatusUnveiling,
			Eyes:      []string{"disintegration"},
			Target:    "api-server",
			Namespace: "default",
			StartTime: time.Now().Truncate(time.Second),
			Duration:  5 * time.Minute,
		},
	}

	if err := store.Save(experiments); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Load() returned %d experiments, want 1", len(loaded))
	}

	if loaded[0].ID != "exp-abc" {
		t.Errorf("ID = %q, want %q", loaded[0].ID, "exp-abc")
	}

	if loaded[0].Status != StatusUnveiling {
		t.Errorf("Status = %q, want %q", loaded[0].Status, StatusUnveiling)
	}
}

func TestStore_Load_MissingFile(t *testing.T) {
	store := tempStore(t)

	experiments, err := store.Load()
	if err != nil {
		t.Fatalf("Load() should not error on missing file, got: %v", err)
	}

	if len(experiments) != 0 {
		t.Errorf("Load() returned %d experiments, want 0", len(experiments))
	}
}

func TestStore_Load_CorruptedJSON(t *testing.T) {
	store := tempStore(t)

	err := os.WriteFile(store.filePath, []byte("not json"), 0o644)
	if err != nil {
		t.Fatalf("writing corrupt file: %v", err)
	}

	_, err = store.Load()
	if err == nil {
		t.Fatal("Load() should error on corrupted JSON")
	}
}

func TestStore_Save_Overwrite(t *testing.T) {
	store := tempStore(t)

	first := []Experiment{{ID: "first", Status: StatusUnveiling}}
	second := []Experiment{{ID: "second", Status: StatusRevealed}}

	_ = store.Save(first)
	_ = store.Save(second)

	loaded, _ := store.Load()
	if len(loaded) != 1 || loaded[0].ID != "second" {
		t.Errorf("Save should overwrite, got %v", loaded)
	}
}

func TestNewStore_CreatesDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	expectedPath := filepath.Join(tmpHome, ".viy", "state.json")
	if store.filePath != expectedPath {
		t.Errorf("filePath = %q, want %q", store.filePath, expectedPath)
	}

	info, err := os.Stat(filepath.Join(tmpHome, ".viy"))
	if err != nil {
		t.Fatalf("state directory should exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("state directory should be a directory")
	}
}

func TestNewStore_IdempotentOnExistingDirectory(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	if err := os.MkdirAll(filepath.Join(tmpHome, ".viy"), 0o700); err != nil {
		t.Fatalf("creating pre-existing dir: %v", err)
	}

	_, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() should succeed when directory already exists: %v", err)
	}
}

func TestStore_SaveAndLoad_RoundTrip(t *testing.T) {
	store := tempStore(t)

	now := time.Now().Truncate(time.Second)
	endTime := now.Add(5 * time.Minute)
	experiments := []Experiment{
		{
			ID:        "exp-1",
			Status:    StatusUnveiling,
			Eyes:      []string{"disintegration"},
			Target:    "api",
			Namespace: "default",
			StartTime: now,
			Duration:  5 * time.Minute,
		},
		{
			ID:        "exp-2",
			Status:    StatusRevealed,
			Eyes:      []string{"disintegration", "charm"},
			Target:    "web",
			Namespace: "staging",
			StartTime: now,
			EndTime:   &endTime,
			Duration:  10 * time.Minute,
		},
	}

	if err := store.Save(experiments); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("Load() returned %d experiments, want 2", len(loaded))
	}

	if loaded[1].Namespace != "staging" {
		t.Errorf("second experiment namespace = %q, want %q", loaded[1].Namespace, "staging")
	}
}

func TestStore_Save_UnwritableDirectory(t *testing.T) {
	store := &Store{
		filePath: "/proc/nonexistent/state.json",
	}

	experiments := []Experiment{{ID: "x"}}

	err := store.Save(experiments)
	if err == nil {
		t.Fatal("Save() should fail on unwritable path")
	}
}

func TestStore_Load_UnreadablePath(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "state.json")

	if err := os.WriteFile(filePath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(filePath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filePath, 0o600) })

	store := &Store{filePath: filePath}

	_, err := store.Load()
	if err == nil {
		t.Fatal("Load() should fail on unreadable file")
	}
}

func tempStore(t *testing.T) *Store {
	t.Helper()

	directory := t.TempDir()
	return &Store{
		filePath: filepath.Join(directory, "state.json"),
	}
}
