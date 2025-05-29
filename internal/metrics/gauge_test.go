package metrics

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

func TestGauge_WorkerFunc(t *testing.T) {
	mp := metric.NewMeterProvider()
	cfg := GaugeConfig{
		Name:        "test.gauge",
		Description: "Test gauge",
		Unit:        "1",
	}
	c := NewConfig()
	c.NumMetrics = 1
	c.ServiceName = "test-service"
	c.Rate = 1
	c.TotalDuration = 10 * time.Millisecond
	logger := zap.NewNop()
	worker := gauge(mp, cfg, *c, logger)
	if worker == nil {
		t.Fatal("expected non-nil WorkerFunc")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	worker(ctx)
}
