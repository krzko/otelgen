package scenarios

import (
	"context"
	"math/rand"
	"os"
	"time"

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

func BasicScenario(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, serviceName string) error {
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

	// Simulate some work for the ping span
	pingDuration := time.Duration(rand.Intn(100)) * time.Millisecond
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

	// Simulate some work for the pong span
	pongDuration := time.Duration(rand.Intn(100)) * time.Millisecond
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
