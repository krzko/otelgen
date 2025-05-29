package metrics

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

func TestExponentialHistogram_WorkerFunc(t *testing.T) {
	mp := metric.NewMeterProvider()
	cfg := ExponentialHistogramConfig{
		Name:          "test.exphistogram",
		Description:   "Test exponential histogram",
		Unit:          "1",
		MaxSize:       10,
		ZeroThreshold: 0.1,
	}
	c := NewConfig()
	c.NumMetrics = 1
	c.ServiceName = "test-service"
	c.Rate = 1
	c.TotalDuration = 10 * time.Millisecond
	logger := zap.NewNop()
	worker := exponentialHistogram(mp, cfg, *c, logger)
	if worker == nil {
		t.Fatal("expected non-nil WorkerFunc")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	worker(ctx)
}
