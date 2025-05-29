// Package logs provides logging configuration and utilities for the trazr-gen application.
package logs

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds configuration for logging.
type Config struct {
	WorkerCount   int
	NumLogs       int
	Rate          float64
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

// HeaderValue represents a map of header key-value pairs for logging.
type HeaderValue map[string]string

var _ flag.Value = (*HeaderValue)(nil)

// Set parses and sets a header value from a string in the form key=value.
func (v *HeaderValue) Set(s string) error {
	kv := strings.SplitN(s, "=", 2)
	if len(kv) != 2 {
		return errors.New("value should be of the format key=value")
	}
	(*v)[kv[0]] = kv[1]
	return nil
}

// String returns the string representation of the HeaderValue.
func (v *HeaderValue) String() string {
	return fmt.Sprintf("%v", *v)
}

// Validate checks the Config for correctness.
func (c *Config) Validate() error {
	if c.Rate < 0 {
		return errors.New("rate must be positive")
	}
	if c.TotalDuration < 0 {
		return errors.New("total duration must be non-negative")
	}
	if c.NumLogs < 0 {
		return errors.New("num logs must be non-negative")
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
	return nil
}

// NewConfig returns a Config with sensible defaults.
func NewConfig() *Config {
	return &Config{
		ServiceName:   "trazr-gen",
		WorkerCount:   1,
		NumLogs:       0,
		Rate:          0,
		TotalDuration: 0,
		Output:        "terminal",
		Insecure:      false,
		UseHTTP:       false,
		Headers:       HeaderValue{},
		Attributes:    []string{},
	}
}
