package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExperimentStatus represents the lifecycle state of an experiment.
type ExperimentStatus string

const (
	StatusUnveiling ExperimentStatus = "unveiling"
	StatusPaused    ExperimentStatus = "paused"
	StatusRevealed  ExperimentStatus = "revealed"
	StatusFailed    ExperimentStatus = "failed"
)

// Experiment holds the persisted state of a single experiment.
type Experiment struct {
	ID             string           `json:"id"`
	Status         ExperimentStatus `json:"status"`
	Eyes           []string         `json:"eyes"`
	Target         string           `json:"target"`
	Namespace      string           `json:"namespace"`
	StartTime      time.Time        `json:"startTime"`
	EndTime        *time.Time       `json:"endTime,omitempty"`
	Duration       time.Duration    `json:"duration"`
	AutoRollbackAt *time.Time       `json:"autoRollbackAt,omitempty"`
}

// Store persists experiments to a local JSON file.
type Store struct {
	filePath string
}

// NewStore creates a Store writing to ~/.viy/state.json.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}

	stateDirectory := filepath.Join(home, ".viy")
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		return nil, fmt.Errorf("creating state directory: %w", err)
	}

	return &Store{
		filePath: filepath.Join(stateDirectory, "state.json"),
	}, nil
}

// NewTestStore creates a Store with an explicit file path for testing.
func NewTestStore(filePath string) *Store {
	return &Store{filePath: filePath}
}

// Save writes the given experiments to disk.
func (s *Store) Save(experiments []Experiment) error {
	data, err := json.MarshalIndent(experiments, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling state: %w", err)
	}

	tmpPath := s.filePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	return os.Rename(tmpPath, s.filePath)
}

// Load reads experiments from disk. Returns an empty slice when no file
// exists yet.
func (s *Store) Load() ([]Experiment, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Experiment{}, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var experiments []Experiment
	if err := json.Unmarshal(data, &experiments); err != nil {
		return nil, fmt.Errorf("unmarshalling state: %w", err)
	}

	return experiments, nil
}
