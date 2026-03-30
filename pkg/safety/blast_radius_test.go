package safety

import (
	"errors"
	"testing"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestCalculateMaxAffected(t *testing.T) {
	tests := []struct {
		name        string
		total       int
		config      BlastRadiusConfig
		wantMax     int
		wantErr     bool
		wantErrType error
	}{
		{
			name:    "30 percent of 10",
			total:   10,
			config:  BlastRadiusConfig{MaxPercentage: 30, MinHealthyReplicas: 0},
			wantMax: 3,
		},
		{
			name:    "50 percent of 10",
			total:   10,
			config:  BlastRadiusConfig{MaxPercentage: 50, MinHealthyReplicas: 0},
			wantMax: 5,
		},
		{
			name:    "minimum 1 when percentage rounds to 0",
			total:   2,
			config:  BlastRadiusConfig{MaxPercentage: 10, MinHealthyReplicas: 0},
			wantMax: 1,
		},
		{
			name:    "respects min healthy replicas",
			total:   10,
			config:  BlastRadiusConfig{MaxPercentage: 60, MinHealthyReplicas: 2},
			wantMax: 6,
		},
		{
			name:        "violates min healthy replicas",
			total:       3,
			config:      BlastRadiusConfig{MaxPercentage: 80, MinHealthyReplicas: 3},
			wantErr:     true,
			wantErrType: viyerrors.ErrBlastRadiusExceeded,
		},
		{
			name:    "100 percent",
			total:   5,
			config:  BlastRadiusConfig{MaxPercentage: 100, MinHealthyReplicas: 0},
			wantMax: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateMaxAffected(tt.total, tt.config)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
					t.Errorf("error = %v, want %v", err, tt.wantErrType)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.wantMax {
				t.Errorf("CalculateMaxAffected() = %d, want %d", got, tt.wantMax)
			}
		})
	}
}
