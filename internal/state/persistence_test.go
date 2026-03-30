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

func tempStore(t *testing.T) *Store {
	t.Helper()

	directory := t.TempDir()
	return &Store{
		filePath: filepath.Join(directory, "state.json"),
	}
}
