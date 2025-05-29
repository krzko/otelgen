// Package scenarios provides tracing scenario implementations for testing and demonstration purposes.
package scenarios

import (
	"context"
	"os"
	"time"

	"github.com/medxops/trazr-gen/internal/attributes"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	fakeIP  string = "1.2.3.4"
	fakeNS  string = "Demo"
	fakeVer string = "1.2.3"
)

// BasicScenario simulates a basic client-server trace scenario for testing and demonstration purposes.
func BasicScenario(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, _ string, attributesList []string) error {
	hn, _ := os.Hostname()

	ctx, sp := tracer.Start(ctx, "ping",
		trace.WithAttributes(
			attribute.String("span.kind", "client"),
			semconv.ServiceNamespace(fakeNS),
			semconv.NetworkPeerAddress(fakeIP),
			semconv.PeerServiceKey.String("ping-pong-server"),
			semconv.ServiceInstanceIDKey.String(hn),
			semconv.ServiceVersionKey.String(fakeVer),
			semconv.TelemetrySDKLanguageGo,
		),
	)
	defer sp.End()

	// Inject sensitive data if the 'sensitive' attribute is present
	if attributes.HasAttribute(attributesList, "sensitive") {
		attributes.InjectRandomSensitiveAttributes(sp, attributesList)
	}

	// Use a local rand.Rand instance
	r := NewRand()

	// Simulate some work for the ping span
	pingDuration := time.Duration(r.IntN(100)) * time.Millisecond
	time.Sleep(pingDuration)

	_, child := tracer.Start(ctx, "pong",
		trace.WithAttributes(
			attribute.String("span.kind", "server"),
			semconv.ServiceNamespace(fakeNS),
			semconv.NetworkPeerAddress(fakeIP),
			semconv.PeerServiceKey.String("ping-pong-client"),
			semconv.ServiceInstanceIDKey.String(hn),
			semconv.ServiceVersionKey.String(fakeVer),
			semconv.TelemetrySDKLanguageGo,
		),
	)

	// Inject sensitive data if the 'sensitive' attribute is present
	if attributes.HasAttribute(attributesList, "sensitive") {
		attributes.InjectRandomSensitiveAttributes(child, attributesList)
	}

	// Simulate some work for the pong span
	pongDuration := time.Duration(r.IntN(100)) * time.Millisecond
	time.Sleep(pongDuration)

	child.End()

	logger.Info("Trace",
		zap.String("traceId", sp.SpanContext().TraceID().String()),
		zap.String("parentSpanId", sp.SpanContext().SpanID().String()),
		zap.String("childSpanId", child.SpanContext().SpanID().String()),
		zap.Duration("pingDuration", pingDuration),
		zap.Duration("pongDuration", pongDuration),
	)

	return nil
}
