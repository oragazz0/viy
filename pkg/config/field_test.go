package config

import (
	"errors"
	"testing"
	"time"

	viyerrors "github.com/oragazz0/viy/pkg/errors"
)

func TestIntField(t *testing.T) {
	cases := []struct {
		name    string
		raw     map[string]any
		want    int
		wantErr bool
	}{
		{"missing returns zero", map[string]any{}, 0, false},
		{"int value", map[string]any{"count": 3}, 3, false},
		{"int64 value", map[string]any{"count": int64(5)}, 5, false},
		{"float64 value (YAML default)", map[string]any{"count": float64(7)}, 7, false},
		{"string value errors", map[string]any{"count": "three"}, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IntField(tc.raw, "count")
			if (err != nil) != tc.wantErr {
				t.Fatalf("IntField err = %v, wantErr %t", err, tc.wantErr)
			}

			if err != nil && !errors.Is(err, viyerrors.ErrInvalidConfiguration) {
				t.Errorf("err should wrap ErrInvalidConfiguration, got %v", err)
			}

			if got != tc.want {
				t.Errorf("IntField = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDurationField(t *testing.T) {
	cases := []struct {
		name    string
		raw     map[string]any
		want    time.Duration
		wantErr bool
	}{
		{"missing returns zero", map[string]any{}, 0, false},
		{"string '5m'", map[string]any{"dur": "5m"}, 5 * time.Minute, false},
		{"string '200ms'", map[string]any{"dur": "200ms"}, 200 * time.Millisecond, false},
		{"unparseable errors", map[string]any{"dur": "nope"}, 0, true},
		{"wrong type errors", map[string]any{"dur": 123}, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DurationField(tc.raw, "dur")
			if (err != nil) != tc.wantErr {
				t.Fatalf("DurationField err = %v, wantErr %t", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("DurationField = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFloatField(t *testing.T) {
	value, err := FloatField(map[string]any{"ratio": 0.5}, "ratio")
	if err != nil {
		t.Fatalf("FloatField err = %v", err)
	}

	if value != 0.5 {
		t.Errorf("FloatField = %v, want 0.5", value)
	}

	intValue, err := FloatField(map[string]any{"ratio": 42}, "ratio")
	if err != nil {
		t.Fatalf("FloatField from int err = %v", err)
	}

	if intValue != 42 {
		t.Errorf("FloatField from int = %v, want 42", intValue)
	}
}

func TestStringField_Missing(t *testing.T) {
	got, err := StringField(map[string]any{}, "missing")
	if err != nil {
		t.Fatalf("StringField err = %v", err)
	}

	if got != "" {
		t.Errorf("StringField missing = %q, want empty", got)
	}
}

func TestStringField_WrongType(t *testing.T) {
	_, err := StringField(map[string]any{"key": 123}, "key")
	if err == nil {
		t.Fatal("StringField should fail on non-string value")
	}
}
