package metrics

import (
	"fmt"
	"math/rand/v2"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Exemplar represents a sample data point with attributes for metrics.
type Exemplar struct {
	FilteredAttributes []attribute.KeyValue
	TimeUnix           int64
	Value              float64
	SpanID             trace.SpanID
	TraceID            trace.TraceID
}

func generateExemplar(r *rand.Rand, value float64, timestamp time.Time) Exemplar {
	return Exemplar{
		FilteredAttributes: []attribute.KeyValue{
			attribute.String("exemplar_attribute", fmt.Sprintf("value-%d", r.IntN(100))),
		},
		TimeUnix: timestamp.UnixNano(),
		Value:    value,
		SpanID:   generateSpanID(r),
		TraceID:  generateTraceID(r),
	}
}

func generateSpanID(r *rand.Rand) trace.SpanID {
	var spanID trace.SpanID
	for i := range spanID {
		spanID[i] = byte(r.IntN(256))
	}
	return spanID
}

func generateTraceID(r *rand.Rand) trace.TraceID {
	var traceID trace.TraceID
	for i := range traceID {
		traceID[i] = byte(r.IntN(256))
	}
	return traceID
}
