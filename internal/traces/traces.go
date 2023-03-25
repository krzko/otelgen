package traces

import (
	"context"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type worker struct {
	running          *atomic.Bool    // pointer to shared flag that indicates it's time to stop the test
	numTraces        int             // how many traces the worker has to generate (only when duration==0)
	propagateContext bool            // whether the worker needs to propagate the trace context via HTTP headers
	totalDuration    time.Duration   // how long to run the test for (overrides `numTraces`)
	limitPerSecond   rate.Limit      // how many spans per second to generate
	wg               *sync.WaitGroup // notify when done
	logger           *zap.Logger
}

const (
	fakeIP  string = "1.2.3.4"
	fakeNS  string = "Demo"
	fakeVer string = "1.2.3"

	fakeSpanDuration = 1234 * time.Millisecond
)

func (w worker) simulateTraces(sn string) {
	tracer := otel.Tracer(sn)
	limiter := rate.NewLimiter(w.limitPerSecond, 1)
	var i int
	hn, _ := os.Hostname()
	for w.running.Load() {
		w.logger.Info("starting traces")
		ctx, sp := tracer.Start(context.Background(), "ping", trace.WithAttributes(
			attribute.String("span.kind", "client"), // is there a semantic convention for this?
			semconv.ServiceNamespace(fakeNS),
			semconv.NetPeerName(fakeIP),
			semconv.PeerServiceKey.String(sn+"-server"),
			semconv.ServiceInstanceIDKey.String(hn),
			semconv.ServiceVersionKey.String(fakeVer),
			semconv.TelemetrySDKLanguageGo,
		))

		childCtx := ctx
		if w.propagateContext {
			header := propagation.HeaderCarrier{}
			// simulates going remote
			otel.GetTextMapPropagator().Inject(childCtx, header)

			// simulates getting a request from a client
			childCtx = otel.GetTextMapPropagator().Extract(childCtx, header)
		}

		_, child := tracer.Start(childCtx, "pong", trace.WithAttributes(
			attribute.String("span.kind", "server"),
			semconv.ServiceNamespace(fakeNS),
			semconv.NetPeerName(fakeIP),
			semconv.PeerServiceKey.String(sn+"-client"),
			semconv.ServiceInstanceIDKey.String(hn),
			semconv.ServiceVersionKey.String(fakeVer),
			semconv.TelemetrySDKLanguageGo,
		))

		if err := limiter.Wait(context.Background()); err != nil {
			w.logger.Fatal("limiter waited failed, retry", zap.Error(err))
		}

		opt := trace.WithTimestamp(time.Now().Add(fakeSpanDuration))
		w.logger.Info("Trace", zap.String("traceId", sp.SpanContext().TraceID().String()))
		w.logger.Info("Parent Span", zap.String("spanId", sp.SpanContext().SpanID().String()))
		w.logger.Info("Child Span", zap.String("spanId", child.SpanContext().SpanID().String()))
		child.End(opt)
		sp.End(opt)

		i++
		if w.numTraces != 0 {
			if i >= w.numTraces {
				break
			}
		}
	}
	w.logger.Info("traces generated", zap.Int("traces", i))
	w.wg.Done()
}
