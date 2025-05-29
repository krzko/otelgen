package metrics

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

func TestHistogram_WorkerFunc(t *testing.T) {
	mp := metric.NewMeterProvider()
	cfg := HistogramConfig{
		Name:        "test.histogram",
		Description: "Test histogram",
		Unit:        "1",
		Bounds:      []float64{0, 10, 20},
	}
	c := NewConfig()
	c.NumMetrics = 1
	c.ServiceName = "test-service"
	c.Rate = 1
	c.TotalDuration = 10 * time.Millisecond
	logger := zap.NewNop()
	worker := histogram(mp, cfg, *c, logger)
	if worker == nil {
		t.Fatal("expected non-nil WorkerFunc")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	worker(ctx)
}
