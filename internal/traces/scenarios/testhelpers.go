//go:build test || !integration
// +build test !integration

package scenarios

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// DummyTracer is a test helper that implements the trace.Tracer interface for testing.
type DummyTracer struct{ trace.Tracer }

// Start implements the trace.Tracer interface for DummyTracer.
func (d DummyTracer) Start(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
	return ctx, &DummySpan{}
}

// DummySpan is a test helper that implements the trace.Span interface for testing.
type DummySpan struct{ trace.Span }

// End implements the trace.Span interface for DummySpan.
func (d *DummySpan) End(_ ...trace.SpanEndOption) {}

// SpanContext implements the trace.Span interface for DummySpan.
func (d *DummySpan) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{})
}

// AddEvent implements the trace.Span interface for DummySpan.
func (d *DummySpan) AddEvent(_ string, _ ...trace.EventOption) {}

// SetAttributes implements the trace.Span interface for DummySpan.
func (d *DummySpan) SetAttributes(_ ...attribute.KeyValue) {}

// SetStatus implements the trace.Span interface for DummySpan.
func (d *DummySpan) SetStatus(_ codes.Code, _ string) {}

// RecordError implements the trace.Span interface for DummySpan.
func (d *DummySpan) RecordError(_ error, _ ...trace.EventOption) {}

// AddLink implements the trace.Span interface for DummySpan.
func (d *DummySpan) AddLink(_ trace.Link) {}
