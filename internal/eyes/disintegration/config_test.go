package disintegration

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid kill count",
			config:  Config{PodKillCount: 3, Strategy: "random"},
			wantErr: false,
		},
		{
			name:    "valid kill percentage",
			config:  Config{PodKillPercentage: 30, Strategy: "sequential"},
			wantErr: false,
		},
		{
			name:    "default strategy accepted",
			config:  Config{PodKillCount: 1},
			wantErr: false,
		},
		{
			name:    "with interval and grace period",
			config:  Config{PodKillCount: 2, Interval: 30 * time.Second, GracePeriod: 5 * time.Second},
			wantErr: false,
		},
		{
			name:    "zero count and zero percentage",
			config:  Config{},
			wantErr: true,
		},
		{
			name:    "both count and percentage set",
			config:  Config{PodKillCount: 3, PodKillPercentage: 30},
			wantErr: true,
		},
		{
			name:    "percentage over 100",
			config:  Config{PodKillPercentage: 150},
			wantErr: true,
		},
		{
			name:    "invalid strategy",
			config:  Config{PodKillCount: 1, Strategy: "worst-first"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
