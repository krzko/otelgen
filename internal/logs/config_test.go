package logs

import (
	"testing"
)

func TestHeaderValue_Set_Valid(t *testing.T) {
	h := HeaderValue{}
	err := h.Set("foo=bar")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if h["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %v", h)
	}
}

func TestHeaderValue_Set_Invalid(t *testing.T) {
	h := HeaderValue{}
	err := h.Set("foobar")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestHeaderValue_String(t *testing.T) {
	h := HeaderValue{"foo": "bar"}
	want := "map[foo:bar]"
	if h.String() != want {
		t.Errorf("expected %q, got %q", want, h.String())
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{"valid", &Config{Rate: 1, TotalDuration: 1, NumLogs: 1, ServiceName: "svc", Output: "out"}, false},
		{"negative rate", &Config{Rate: -1, TotalDuration: 1, NumLogs: 1, ServiceName: "svc", Output: "out"}, true},
		{"negative duration", &Config{Rate: 1, TotalDuration: -1, NumLogs: 1, ServiceName: "svc", Output: "out"}, true},
		{"negative num logs", &Config{Rate: 1, TotalDuration: 1, NumLogs: -1, ServiceName: "svc", Output: "out"}, true},
		{"empty service name", &Config{Rate: 1, TotalDuration: 1, NumLogs: 1, ServiceName: "", Output: "out"}, true},
		{"empty output", &Config{Rate: 1, TotalDuration: 1, NumLogs: 1, ServiceName: "svc", Output: ""}, true},
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
