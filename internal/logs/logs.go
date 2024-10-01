package logs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Run initialises log generation based on the provided configuration.
func Run(c *Config, logger *zap.Logger) error {
	logger.Debug("Log generation config", zap.Any("Config", c))

	if c.NumLogs == 0 && c.TotalDuration == 0 {
		// Log without using zap.Error, which logs stack traces
		logger.Warn("No log number or duration specified. Log generation will continue indefinitely.")
	}

	// Configure rate limiter
	limit := rate.Limit(c.Rate)
	if c.Rate == 0 {
		limit = rate.Inf
		logger.Info("Generation of logs isn't being throttled")
	} else {
		logger.Info("Generation of logs is limited", zap.Float64("per-second", float64(limit)))
	}

	// Create OTLP exporter
	exporter, err := createExporter(c)
	if err != nil {
		// Log the error as a string without the stack trace
		logger.Error("Failed to create exporter", zap.String("error", err.Error()))
		return fmt.Errorf("failed to create exporter: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := exporter.Shutdown(ctx); err != nil {
			// Log the error as a string without the stack trace
			logger.Error("Failed to shutdown exporter", zap.String("error", err.Error()))
		}
	}()

	// Define resource attributes
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.ServiceName),
		semconv.K8SNamespaceNameKey.String("default"),
		semconv.K8SContainerNameKey.String("otelgen"),
		semconv.K8SPodNameKey.String(generatePodName()),
		semconv.HostNameKey.String("node-1"),
	)
	logger.Debug("Resource attributes set", zap.String("Resource", res.String()))

	// Set up a BatchProcessor and pass it to the LoggerProvider
	batchProcessor := sdklog.NewBatchProcessor(exporter,
		sdklog.WithMaxQueueSize(2048),
		sdklog.WithExportMaxBatchSize(512),
		sdklog.WithExportInterval(1*time.Second),
	)

	// Initialise LoggerProvider with BatchProcessor and Resource
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(batchProcessor),
		sdklog.WithResource(res),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := loggerProvider.Shutdown(ctx); err != nil {
			// Log the error as a string without the stack trace
			logger.Error("Failed to shutdown logger provider", zap.String("error", err.Error()))
		}
	}()

	// Initialise wait group for workers
	wg := sync.WaitGroup{}
	running := &atomic.Bool{}
	running.Store(true)

	totalLogs := atomic.Int64{}

	logger.Debug("Worker count", zap.Int("WorkerCount", c.WorkerCount))

	for i := 0; i < c.WorkerCount; i++ {
		wg.Add(1)
		logger.Debug("Starting worker", zap.Int("Worker", i))
		go generateLogs(c, loggerProvider, limit, logger.With(zap.Int("worker", i)), &wg, res, running, &totalLogs)
	}

	// Handle total duration if specified, otherwise run indefinitely
	if c.TotalDuration > 0 {
		time.Sleep(c.TotalDuration)
		running.Store(false)
	}

	// Wait for all workers to finish
	wg.Wait()

	// Log the total number of logs generated
	logger.Info("Log generation completed", zap.Int64("total_logs", totalLogs.Load()))
	return nil
}

// createExporter initialises the OTLP exporter based on the configuration.
func createExporter(c *Config) (sdklog.Exporter, error) {
	ctx := context.Background()
	var exp sdklog.Exporter
	var err error

	if c.UseHTTP {
		opts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(c.Endpoint),
		}
		if c.Insecure {
			opts = append(opts, otlploghttp.WithInsecure())
		}
		if len(c.Headers) > 0 {
			opts = append(opts, otlploghttp.WithHeaders(c.Headers))
		}
		exp, err = otlploghttp.New(ctx, opts...)
	} else {
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(c.Endpoint),
		}
		if c.Insecure {
			opts = append(opts, otlploggrpc.WithInsecure())
		}
		if len(c.Headers) > 0 {
			opts = append(opts, otlploggrpc.WithHeaders(c.Headers))
		}
		exp, err = otlploggrpc.New(ctx, opts...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	return exp, nil
}

// generateLogs handles the log generation for a single worker.
// generateLogs handles the log generation for a single worker.
func generateLogs(c *Config, loggerProvider *sdklog.LoggerProvider, limit rate.Limit, logger *zap.Logger, wg *sync.WaitGroup, res *resource.Resource, running *atomic.Bool, totalLogs *atomic.Int64) {
	defer wg.Done()

	limiter := rate.NewLimiter(limit, 1)
	otelLogger := loggerProvider.Logger(c.ServiceName)

	for i := 0; c.NumLogs == 0 || i < c.NumLogs; i++ {
		if !running.Load() {
			break
		}

		// Only log every 10th log entry
		if i%10 == 0 {
			logger.Debug("Generating log", zap.Int("log_index", i))
		}

		traceID := generateTraceID()
		spanID := generateSpanID()

		// Simulate the web request phases: start, processing, finish
		logPhases := []string{"start", "processing", "finish"}
		httpMethods := []string{"GET", "POST", "PUT", "DELETE"}
		httpMethod := httpMethods[cryptoRandIntn(len(httpMethods))]

		for _, phase := range logPhases {
			phaseDuration := randomDuration(100, 500)

			// Randomize severity and text
			severity, severityText := randomSeverity()

			record := log.Record{}
			record.SetTimestamp(time.Now())
			record.SetObservedTimestamp(time.Now())
			record.SetSeverity(severity)
			record.SetSeverityText(severityText)
			record.SetBody(log.StringValue(fmt.Sprintf("Log %d: %s phase: %s", i, severityText, phase)))

			attrs := []log.KeyValue{
				log.String("worker_id", fmt.Sprintf("%d", i)),
				log.String("service.name", c.ServiceName),
				log.String("trace_id", traceID.String()),
				log.String("span_id", spanID.String()),
				log.String("trace_flags", "01"),
				log.String("phase", phase),
				log.String("http.method", httpMethod),
				log.Int("http.status_code", randomHTTPStatusCode()),
				log.String("http.target", fmt.Sprintf("/api/v1/resource/%d", i)),
				log.String("k8s.pod.name", generatePodName()),
				log.String("k8s.namespace.name", "default"),
				log.String("k8s.container.name", "otelgen"),
			}
			record.AddAttributes(attrs...)

			// Emit the log record
			otelLogger.Emit(context.Background(), record)

			// Simulate the time spent in each phase
			time.Sleep(phaseDuration)

			// Generate a new span ID for each phase
			spanID = generateSpanID()
		}

		totalLogs.Add(int64(len(logPhases)))

		if err := limiter.Wait(context.Background()); err != nil {
			logger.Error("failed to wait for rate limiter", zap.Error(err))
			continue
		}
	}

	logger.Debug("Worker completed log generation", zap.Int64("total_logs", totalLogs.Load()))
}

// generateTraceID generates a new trace ID using crypto/rand.
func generateTraceID() trace.TraceID {
	var tid [16]byte
	_, err := rand.Read(tid[:])
	if err != nil {
		panic(fmt.Sprintf("failed to generate trace ID: %v", err))
	}
	return trace.TraceID(tid)
}

// generateSpanID generates a new span ID using crypto/rand.
func generateSpanID() trace.SpanID {
	var sid [8]byte
	_, err := rand.Read(sid[:])
	if err != nil {
		panic(fmt.Sprintf("failed to generate span ID: %v", err))
	}
	return trace.SpanID(sid)
}

// randomDuration generates a random duration between min and max milliseconds using crypto/rand.
func randomDuration(minMs int, maxMs int) time.Duration {
	diff := maxMs - minMs
	randVal := cryptoRandIntn(diff)
	return time.Duration(minMs+randVal) * time.Millisecond
}

// randomHTTPStatusCode generates a random HTTP status code using crypto/rand.
func randomHTTPStatusCode() int {
	httpStatusCodes := []int{200, 201, 202, 400, 401, 403, 404, 500, 503}
	return httpStatusCodes[cryptoRandIntn(len(httpStatusCodes))]
}

// generatePodName simulates a unique pod name using crypto/rand.
func generatePodName() string {
	podNameSuffix := make([]byte, 4)
	_, _ = rand.Read(podNameSuffix)
	return fmt.Sprintf("otelgen-pod-%s", hex.EncodeToString(podNameSuffix))
}

// randomSeverity generates a random severity level and text.
func randomSeverity() (log.Severity, string) {
	severities := []struct {
		level log.Severity
		text  string
	}{
		{log.SeverityTrace1, "Trace"},
		{log.SeverityDebug, "Debug"},
		{log.SeverityInfo, "Info"},
		{log.SeverityWarn, "Warn"},
		{log.SeverityError, "Error"},
		{log.SeverityFatal, "Fatal"},
	}
	randomIdx := cryptoRandIntn(len(severities))
	return severities[randomIdx].level, severities[randomIdx].text
}

// cryptoRandIntn generates a crypto-random number within the range 0 to max-1.
func cryptoRandIntn(max int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(fmt.Sprintf("failed to generate random number: %v", err))
	}
	return int(nBig.Int64())
}
