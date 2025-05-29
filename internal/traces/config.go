// Package traces provides configuration and utilities for trace generation in trazr-gen.
package traces

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds configuration for trace generation.
type Config struct {
	WorkerCount      int
	NumTraces        int
	PropagateContext bool
	Rate             float64
	TotalDuration    time.Duration
	ServiceName      string
	Scenarios        []string

	// OTLP config
	Output   string
	Insecure bool
	UseHTTP  bool
	Headers  map[string]string

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
		return errors.New("value should be of the format key=value")
	}
	(*v)[kv[0]] = kv[1]
	return nil
}

// String returns the string representation of the HeaderValue map.
func (v *HeaderValue) String() string {
	return fmt.Sprintf("%v", map[string]string(*v))
}

// Validate checks the config for required fields and valid values.
// Note: Zero is allowed for rate, total duration, and num traces
// (meaning "unthrottled" or "indefinite"). Only negative values are invalid.
func (c *Config) Validate() error {
	// Allow zero for unthrottled/indefinite; only negative is invalid
	if c.TotalDuration < 0 {
		return errors.New("total duration must be non-negative")
	}
	if c.NumTraces < 0 {
		return errors.New("num traces must be non-negative")
	}
	if c.WorkerCount < 0 {
		return errors.New("worker count must be positive")
	}
	if c.ServiceName == "" {
		return errors.New("service name must not be empty")
	}
	if c.Output == "" {
		return errors.New("output must not be empty")
	}
	if c.Rate < 0 {
		return errors.New("rate must be non-negative (0 means unthrottled)")
	}
	return nil
}

// NewConfig returns a Config with sensible defaults.
func NewConfig() *Config {
	return &Config{
		ServiceName:   "trazr-gen",
		WorkerCount:   1,
		NumTraces:     3,
		Rate:          0.0,
		TotalDuration: 0,
		Scenarios:     []string{"basic"},
		Output:        "terminal",
		Insecure:      false,
		UseHTTP:       false,
		Headers:       map[string]string{},
		Attributes:    []string{},
	}
}
