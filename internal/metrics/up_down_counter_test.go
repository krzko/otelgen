package metrics

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

func TestUpDownCounter_WorkerFunc(t *testing.T) {
	mp := metric.NewMeterProvider()
	c := NewConfig()
	c.NumMetrics = 1
	c.ServiceName = "test-service"
	c.Rate = 1
	c.TotalDuration = 10 * time.Millisecond
	logger := zap.NewNop()
	worker := upDownCounter(mp, *c, logger)
	if worker == nil {
		t.Fatal("expected non-nil WorkerFunc")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	worker(ctx)
}
