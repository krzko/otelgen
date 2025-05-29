package metrics

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{"valid", &Config{Rate: 1, TotalDuration: 1, NumMetrics: 1, ServiceName: "svc", Output: "out"}, false},
		{"negative rate", &Config{Rate: -1, TotalDuration: 1, NumMetrics: 1, ServiceName: "svc", Output: "out"}, true},
		{"negative duration", &Config{Rate: 1, TotalDuration: -1, NumMetrics: 1, ServiceName: "svc", Output: "out"}, true},
		{"negative num metrics", &Config{Rate: 1, TotalDuration: 1, NumMetrics: -1, ServiceName: "svc", Output: "out"}, true},
		{"empty service name", &Config{Rate: 1, TotalDuration: 1, NumMetrics: 1, ServiceName: "", Output: "out"}, true},
		{"empty output", &Config{Rate: 1, TotalDuration: 1, NumMetrics: 1, ServiceName: "svc", Output: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
