// Package metrics provides types and functions for generating synthetic OpenTelemetry metrics.
package metrics

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds configuration for metrics generation.
type Config struct {
	WorkerCount   int
	NumMetrics    int
	Rate          float64 // Metrics per second. 0 means unthrottled. Defaults to 1 if not set.
	TotalDuration time.Duration
	ServiceName   string

	// OTLP config
	Output   string
	Insecure bool
	UseHTTP  bool
	Headers  HeaderValue

	// Scenario attributes from CLI
	Attributes []string
}

// HeaderValue is a map of header key-value pairs for OTLP exporters.
type HeaderValue map[string]string

var _ flag.Value = (*HeaderValue)(nil)

// Set parses a header string in the form key=value and adds it to the HeaderValue map.
func (v *HeaderValue) Set(s string) error {
	kv := strings.SplitN(s, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("invalid header format: %s", s)
	}
	(*v)[kv[0]] = kv[1]
	return nil
}

// String returns the string representation of the HeaderValue map.
func (v *HeaderValue) String() string {
	return fmt.Sprintf("%v", map[string]string(*v))
}

// Validate checks the configuration for required fields and valid values.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}
	if c.WorkerCount < 0 {
		return errors.New("worker count must be positive")
	}
	if c.NumMetrics < 0 {
		return errors.New("num metrics must be non-negative")
	}
	if c.Rate < 0 {
		return errors.New("rate must be positive")
	}
	if c.TotalDuration < 0 {
		return errors.New("total duration must be non-negative")
	}
	if c.ServiceName == "" {
		return errors.New("service name must not be empty")
	}
	if c.Output == "" {
		return errors.New("output must not be empty")
	}
	return nil
}

// NewConfig returns a Config with sensible defaults.
func NewConfig() *Config {
	return &Config{
		ServiceName:   "trazr-gen",
		WorkerCount:   1,
		NumMetrics:    1,
		Rate:          1.0,
		TotalDuration: 0,
		Output:        "terminal",
		Insecure:      false,
		UseHTTP:       false,
		Headers:       HeaderValue{},
		Attributes:    []string{},
	}
}
