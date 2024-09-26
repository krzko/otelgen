package metrics

import (
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

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
			attribute.String("exemplar_attribute", fmt.Sprintf("value-%d", r.Intn(100))),
		},
		TimeUnix: timestamp.UnixNano(),
		Value:    value,
		SpanID:   generateSpanID(r),
		TraceID:  generateTraceID(r),
	}
}

func generateSpanID(r *rand.Rand) trace.SpanID {
	var spanID trace.SpanID
	r.Read(spanID[:])
	return spanID
}

func generateTraceID(r *rand.Rand) trace.TraceID {
	var traceID trace.TraceID
	r.Read(traceID[:])
	return traceID
}
