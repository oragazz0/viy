package config

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDuration_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"string 5m", `"5m"`, 5 * time.Minute, false},
		{"string 200ms", `"200ms"`, 200 * time.Millisecond, false},
		{"numeric nanoseconds", `1000000000`, time.Second, false},
		{"null", `null`, 0, false},
		{"invalid string", `"eternity"`, 0, true},
		{"invalid shape", `{}`, 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tc.input), &d)

			if (err != nil) != tc.wantErr {
				t.Fatalf("UnmarshalJSON err = %v, wantErr %t", err, tc.wantErr)
			}

			if d.ToStd() != tc.want {
				t.Errorf("Duration = %v, want %v", d.ToStd(), tc.want)
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	d := Duration(5 * time.Minute)
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("Marshal err = %v", err)
	}

	if string(data) != `"5m0s"` {
		t.Errorf("Marshal = %s, want %q", data, `"5m0s"`)
	}
}
