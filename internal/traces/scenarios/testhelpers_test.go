package scenarios

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestDummyTracerAndSpan(t *testing.T) {
	tracer := DummyTracer{}
	ctx := context.Background()
	ctx2, span := tracer.Start(ctx, "test-span")
	if ctx2 != ctx {
		t.Error("expected context to be unchanged")
	}
	span.End()
	_ = span.SpanContext()
	span.AddEvent("event")
	span.SetAttributes()
	span.SetStatus(0, "")
	span.RecordError(nil)
	span.AddLink(trace.Link{})
}
