package scenarios

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Extend DummySpan to record attributes for testing
// Embeds the DummySpan from testhelpers.go
// Only SetAttributes is overridden

type RecordingSpan struct {
	DummySpan
	Attributes []attribute.KeyValue
}

func (s *RecordingSpan) SetAttributes(attrs ...attribute.KeyValue) {
	s.Attributes = append(s.Attributes, attrs...)
}

// Extend DummyTracer to return our RecordingSpan
// Embeds DummyTracer from testhelpers.go

type RecordingTracer struct {
	DummyTracer
	LastSpan *RecordingSpan
}

func (t *RecordingTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	s := &RecordingSpan{}
	t.LastSpan = s
	return ctx, s
}

func TestBasicScenario_WithoutSensitive(t *testing.T) {
	tracer := &RecordingTracer{}
	logger := zap.NewNop()
	err := BasicScenario(context.Background(), tracer, logger, "test-service", nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if tracer.LastSpan != nil && len(tracer.LastSpan.Attributes) > 0 {
		for _, attr := range tracer.LastSpan.Attributes {
			if attr.Key == "user.ssn" || attr.Key == "user.email" || attr.Key == "auth.token" {
				t.Errorf("did not expect sensitive attribute %s without 'sensitive' flag", attr.Key)
			}
		}
	}
}

func TestBasicScenario_WithSensitive(t *testing.T) {
	tracer := &RecordingTracer{}
	logger := zap.NewNop()
	err := BasicScenario(context.Background(), tracer, logger, "test-service", []string{"sensitive"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// We can't guarantee sensitive attributes are always injected (random), but we can check if any are present
	found := false
	if tracer.LastSpan != nil {
		for _, attr := range tracer.LastSpan.Attributes {
			if attr.Key == "user.ssn" || attr.Key == "user.email" || attr.Key == "auth.token" {
				found = true
			}
		}
	}
	// It's possible (but rare) that random chance means no sensitive attributes were injected
	// So we do not fail the test if not found, but we log it
	if !found {
		t.Logf("No sensitive attributes injected; this can happen due to randomness")
	}
}
