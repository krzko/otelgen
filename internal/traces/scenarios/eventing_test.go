package scenarios

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestEventingScenario_NoError(t *testing.T) {
	tracer := DummyTracer{}
	logger := zap.NewNop()
	err := EventingScenario(context.Background(), tracer, logger, "test-service", nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
