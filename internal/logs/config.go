package logs

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Config struct {
	WorkerCount   int
	NumLogs       int
	Rate          float64
	TotalDuration time.Duration
	ServiceName   string

	// OTLP config
	Endpoint string
	Insecure bool
	UseHTTP  bool
	Headers  HeaderValue
}

type HeaderValue map[string]string

var _ flag.Value = (*HeaderValue)(nil)

func (v *HeaderValue) String() string {
	return ""
}

func (v *HeaderValue) Set(s string) error {
	kv := strings.SplitN(s, "=", 2)
	if len(kv) != 2 {
		return fmt.Errorf("value should be of the format key=value")
	}
	(*v)[kv[0]] = kv[1]
	return nil
}
