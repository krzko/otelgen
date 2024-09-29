package traces

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/krzko/otelgen/internal/traces/scenarios"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type worker struct {
	running          *atomic.Bool
	numTraces        int
	propagateContext bool
	totalDuration    time.Duration
	limitPerSecond   rate.Limit
	wg               *sync.WaitGroup
	logger           *zap.Logger
	scenarios        []string
	serviceName      string
}

func Run(c *Config, logger *zap.Logger) error {
	if c.TotalDuration > 0 {
		c.NumTraces = 0
	} else if c.NumTraces <= 0 {
		return fmt.Errorf("either `traces` or `duration` must be greater than 0")
	}

	limit := rate.Limit(c.Rate)
	if c.Rate == 0 {
		limit = rate.Inf
		logger.Info("generation of traces isn't being throttled")
	} else {
		logger.Info("generation of traces is limited", zap.Float64("per-second", float64(limit)))
	}

	wg := sync.WaitGroup{}
	running := atomic.NewBool(true)

	for i := 0; i < c.WorkerCount; i++ {
		wg.Add(1)
		w := worker{
			running:          running,
			numTraces:        c.NumTraces,
			propagateContext: c.PropagateContext,
			totalDuration:    c.TotalDuration,
			limitPerSecond:   limit,
			wg:               &wg,
			logger:           logger.With(zap.Int("worker", i)),
			scenarios:        c.Scenarios,
			serviceName:      c.ServiceName,
		}
		go w.simulateTraces()
	}

	if c.TotalDuration > 0 {
		logger.Info("generation duration", zap.Float64("seconds", c.TotalDuration.Seconds()))
		time.Sleep(c.TotalDuration)
		running.Store(false)
	}

	wg.Wait()
	return nil
}

func (w *worker) simulateTraces() {
	tracer := otel.Tracer(w.serviceName)
	limiter := rate.NewLimiter(w.limitPerSecond, 1)
	var i int

	for w.running.Load() {
		w.logger.Info("starting traces")
		for _, scenario := range w.scenarios {
			w.logger.Info("generating scenario", zap.String("scenario", scenario))

			ctx, sp := tracer.Start(context.Background(), scenario)
			childCtx := ctx
			if w.propagateContext {
				header := propagation.HeaderCarrier{}
				otel.GetTextMapPropagator().Inject(childCtx, header)
				childCtx = otel.GetTextMapPropagator().Extract(childCtx, header)
			}

			err := runScenario(childCtx, scenario, tracer, w.logger, w.serviceName)
			if err != nil {
				w.logger.Error("failed to run scenario", zap.String("scenario", scenario), zap.Error(err))
			}

			if err := limiter.Wait(context.Background()); err != nil {
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
		if w.numTraces != 0 && i >= w.numTraces {
			break
		}
	}

	w.logger.Info("traces generation completed", zap.Int("totalTraces", i))
	w.wg.Done()
}

func runScenario(ctx context.Context, scenario string, tracer trace.Tracer, logger *zap.Logger, serviceName string) error {
	scenarioFunc, ok := Scenarios[scenario]
	if !ok {
		return fmt.Errorf("unknown scenario: %s", scenario)
	}
	return scenarioFunc(ctx, tracer, logger, serviceName)
}

var Scenarios = map[string]func(context.Context, trace.Tracer, *zap.Logger, string) error{
	"basic":         scenarios.BasicScenario,
	"web_mobile":    scenarios.WebMobileScenario,
	"eventing":      scenarios.EventingScenario,
	"microservices": scenarios.MicroservicesScenario,
}
