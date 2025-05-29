package metrics

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	randv2 "math/rand/v2"
	"os"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// StdoutMetricExporter implements metric.Exporter and prints metrics to stdout as JSON.
type StdoutMetricExporter struct{}

// Export implements the metric.Exporter interface for StdoutMetricExporter.
func (e *StdoutMetricExporter) Export(_ context.Context, rm *metricdata.ResourceMetrics) error {
	b, _ := json.MarshalIndent(rm, "", "  ")
	if _, err := os.Stdout.Write(b); err != nil {
		return err
	}
	if _, err := os.Stdout.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

// ForceFlush implements the metric.Exporter interface for StdoutMetricExporter.
func (e *StdoutMetricExporter) ForceFlush(_ context.Context) error { return nil }

// Shutdown implements the metric.Exporter interface for StdoutMetricExporter.
func (e *StdoutMetricExporter) Shutdown(_ context.Context) error { return nil }

// Aggregation implements the metric.Exporter interface for StdoutMetricExporter.
func (e *StdoutMetricExporter) Aggregation(_ sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return nil
}

// Temporality implements the metric.Exporter interface for StdoutMetricExporter.
func (e *StdoutMetricExporter) Temporality(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// NewRand returns a *randv2.Rand seeded with cryptographically secure randomness.
func NewRand() *randv2.Rand {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("failed to seed PRNG: " + err.Error())
	}
	seed := binary.LittleEndian.Uint64(b[:])
	return randv2.New(randv2.NewPCG(seed, 0))
}
