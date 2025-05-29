package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/medxops/trazr-gen/internal/traces/scenarios"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type worker struct {
	numTraces        int
	propagateContext bool
	totalDuration    time.Duration
	limitPerSecond   rate.Limit
	wg               *sync.WaitGroup
	logger           *zap.Logger
	scenarios        []string
	serviceName      string
	attributes       []string
}

// StdoutSpanExporter implements sdktrace.SpanExporter and prints spans to stdout as JSON.
type StdoutSpanExporter struct{}

// ExportSpans implements the sdktrace.SpanExporter interface for StdoutSpanExporter.
func (e *StdoutSpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		m := map[string]any{
			"name":       span.Name(),
			"trace_id":   span.SpanContext().TraceID().String(),
			"span_id":    span.SpanContext().SpanID().String(),
			"parent_id":  span.Parent().SpanID().String(),
			"start":      span.StartTime().Format(time.RFC3339Nano),
			"end":        span.EndTime().Format(time.RFC3339Nano),
			"attributes": span.Attributes(),
			"status":     span.Status().Code.String(),
		}
		b, _ := json.MarshalIndent(m, "", "  ")
		if _, err := os.Stdout.Write(b); err != nil {
			return err
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown implements the sdktrace.SpanExporter interface for StdoutSpanExporter.
func (e *StdoutSpanExporter) Shutdown(_ context.Context) error { return nil }

// ForceFlush implements the sdktrace.SpanExporter interface for StdoutSpanExporter.
func (e *StdoutSpanExporter) ForceFlush(_ context.Context) error { return nil }

// Run initializes and executes trace generation based on the provided configuration and logger.
func Run(c *Config, logger *zap.Logger) error {
	if err := c.Validate(); err != nil {
		logger.Error("invalid config", zap.Error(err))
		return err
	}

	if c.NumTraces == 0 && c.TotalDuration == 0 {
		logger.Warn("No trace number or duration specified. Trace generation will continue indefinitely.")
	}

	limit := rate.Limit(c.Rate)
	if c.Rate == 0 {
		limit = rate.Inf
		logger.Info("generation of traces isn't being throttled")
	} else {
		logger.Info("generation of traces is limited", zap.Float64("per-second", float64(limit)))
	}

	wg := sync.WaitGroup{}

	// Create a context with timeout for duration-based cancellation
	ctx := context.Background()
	var cancel context.CancelFunc
	if c.TotalDuration > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.TotalDuration)
		defer cancel()
	}

	for i := 0; i < c.WorkerCount; i++ {
		wg.Add(1)
		w := worker{
			numTraces:        c.NumTraces,
			propagateContext: c.PropagateContext,
			totalDuration:    c.TotalDuration,
			limitPerSecond:   limit,
			wg:               &wg,
			logger:           logger.With(zap.Int("worker", i)),
			scenarios:        c.Scenarios,
			serviceName:      c.ServiceName,
			attributes:       c.Attributes,
		}
		go w.simulateTraces(ctx)
	}

	// Wait for all workers to finish (they should exit when ctx is done)
	wg.Wait()
	return nil
}

func (w *worker) simulateTraces(ctx context.Context) {
	tracer := otel.Tracer(w.serviceName)
	limiter := rate.NewLimiter(w.limitPerSecond, 1)
	var i int

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping traces generation due to context cancellation", zap.Int("totalTraces", i))
			w.wg.Done()
			return
		default:
		}

		w.logger.Info("starting traces")
		for _, scenario := range w.scenarios {
			w.logger.Info("generating scenario", zap.String("scenario", scenario))

			ctxSpan, sp := tracer.Start(ctx, scenario)
			childCtx := ctxSpan
			if w.propagateContext {
				header := propagation.HeaderCarrier{}
				otel.GetTextMapPropagator().Inject(childCtx, header)
				childCtx = otel.GetTextMapPropagator().Extract(childCtx, header)
			}

			err := runScenario(childCtx, scenario, tracer, w.logger, w.serviceName, w.attributes)
			if err != nil {
				w.logger.Error("failed to run scenario", zap.String("scenario", scenario), zap.Error(err))
			}

			if err := limiter.Wait(ctx); err != nil {
				w.logger.Fatal("limiter waited failed, retry", zap.Error(err))
			}

			w.logger.Info("scenario completed",
				zap.String("scenario", scenario),
				zap.String("traceId", sp.SpanContext().TraceID().String()),
				zap.String("spanId", sp.SpanContext().SpanID().String()),
			)
			sp.End()
		}

		i++
		if w.numTraces == 0 || i < w.numTraces {
			break
		}
	}

	w.logger.Info("traces generation completed", zap.Int("totalTraces", i))
	w.wg.Done()
}

func runScenario(ctx context.Context, scenario string, tracer trace.Tracer, logger *zap.Logger, serviceName string, attributes []string) error {
	scenarioFunc, ok := Scenarios[scenario]
	if !ok {
		return fmt.Errorf("unknown scenario: %s", scenario)
	}
	return scenarioFunc(ctx, tracer, logger, serviceName, attributes)
}

// Scenarios maps scenario names to their corresponding trace scenario functions.
var Scenarios = map[string]func(context.Context, trace.Tracer, *zap.Logger, string, []string) error{
	"basic":         scenarios.BasicScenario,
	"web_mobile":    scenarios.WebMobileScenario,
	"eventing":      scenarios.EventingScenario,
	"microservices": scenarios.MicroservicesScenario,
}
