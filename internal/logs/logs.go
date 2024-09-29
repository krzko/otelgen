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

func cryptoRandIntn(max int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(fmt.Sprintf("failed to generate random number: %v", err))
	}
	return int(nBig.Int64())
}

// Config holds the configuration for log generation.
type Config struct {
	WorkerCount    int
	NumLogs        int
	ServiceName    string
	Endpoint       string
	Insecure       bool
	UseHTTP        bool
	Rate           float64
	TotalDuration  time.Duration
	SeverityText   string
	SeverityNumber int32
}

// Run initialises log generation based on the provided configuration.
func Run(c *Config, logger *zap.Logger) error {
	// Validate configuration
	if c.TotalDuration > 0 {
		c.NumLogs = 0
	} else if c.NumLogs <= 0 {
		return fmt.Errorf("either `NumLogs` or `TotalDuration` must be greater than 0")
	}

	// Configure rate limiter
	limit := rate.Limit(c.Rate)
	if c.Rate == 0 {
		limit = rate.Inf
		logger.Info("generation of logs isn't being throttled")
	} else {
		logger.Info("generation of logs is limited", zap.Float64("per-second", float64(limit)))
	}

	// Create OTLP exporter
	exporter, err := createExporter(c)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := exporter.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown exporter", zap.Error(err))
		}
	}()

	// Define resource attributes, including Kubernetes-related details
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.ServiceName),
		semconv.K8SNamespaceNameKey.String("default"),
		semconv.K8SContainerNameKey.String("otelgen"),
		semconv.K8SPodNameKey.String(generatePodName()),
		semconv.HostNameKey.String("node-1"),
	)

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
			logger.Error("failed to shutdown logger provider", zap.Error(err))
		}
	}()

	// Initialise wait group for workers
	wg := sync.WaitGroup{}
	running := &atomic.Bool{}
	running.Store(true)

	// Parse severity
	severityText, severityNumber, err := parseSeverity(c.SeverityText, c.SeverityNumber)
	if err != nil {
		return err
	}

	totalLogs := atomic.Int64{}

	for i := 0; i < c.WorkerCount; i++ {
		wg.Add(1)
		go generateLogs(c, loggerProvider, limit, logger.With(zap.Int("worker", i)), &wg, res, running, severityText, severityNumber, &totalLogs)
	}

	// Handle total duration if specified
	if c.TotalDuration > 0 {
		time.Sleep(c.TotalDuration)
		running.Store(false)
	}

	// Wait for all workers to finish
	wg.Wait()

	// Log the total number of logs generated
	logger.Info("log generation completed", zap.Int64("total_logs", totalLogs.Load()))
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
			otlploghttp.WithInsecure(),
		}
		exp, err = otlploghttp.New(ctx, opts...)
	} else {
		opts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(c.Endpoint),
			otlploggrpc.WithInsecure(),
		}
		exp, err = otlploggrpc.New(ctx, opts...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	return exp, nil
}

// generateLogs handles the log generation for a single worker.
func generateLogs(c *Config, loggerProvider *sdklog.LoggerProvider, limit rate.Limit, logger *zap.Logger, wg *sync.WaitGroup, res *resource.Resource, running *atomic.Bool, severityText string, severityNumber log.Severity, totalLogs *atomic.Int64) {
	defer wg.Done()

	limiter := rate.NewLimiter(limit, 1)
	otelLogger := loggerProvider.Logger(c.ServiceName)

	// Log statement to indicate that log generation is starting for this worker
	logger.Info("starting log generation", zap.Int64("worker_id", totalLogs.Load()))

	for i := 0; c.NumLogs == 0 || i < c.NumLogs; i++ {
		if !running.Load() {
			break
		}

		// Generate a single trace ID for the request
		traceID := generateTraceID()
		spanID := generateSpanID()

		// Simulate the web request phases: start, processing, finish
		logPhases := []string{"start", "processing", "finish"}
		httpMethods := []string{"GET", "POST", "PUT", "DELETE"}
		httpMethod := httpMethods[cryptoRandIntn(len(httpMethods))]

		// Randomize duration between phases to simulate request timings
		for _, phase := range logPhases {
			phaseDuration := randomDuration(100, 500) // Random duration between 100ms and 500ms

			record := log.Record{}
			record.SetTimestamp(time.Now())
			record.SetObservedTimestamp(time.Now())
			record.SetSeverity(severityNumber)
			record.SetSeverityText(severityText)
			record.SetBody(log.StringValue(fmt.Sprintf("Log %d: %s phase: %s", i, severityText, phase)))

			// Add attributes including trace_id, span_id, and Kubernetes/HTTP attributes
			attrs := []log.KeyValue{
				log.String("worker_id", fmt.Sprintf("%d", i)),
				log.String("service.name", c.ServiceName),
				log.String("trace_id", traceID.String()),
				log.String("span_id", spanID.String()),
				log.String("trace_flags", "01"), // Assuming trace is sampled
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

		// Update the total number of logs
		totalLogs.Add(int64(len(logPhases)))

		if err := limiter.Wait(context.Background()); err != nil {
			logger.Error("failed to wait for rate limiter", zap.Error(err))
			continue
		}
	}
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

// parseSeverity validates and parses severity text and number.
func parseSeverity(severityText string, severityNumber int32) (string, log.Severity, error) {
	sn := log.Severity(severityNumber)
	if sn < log.SeverityTrace1 || sn > log.SeverityFatal4 {
		return "", log.SeverityUndefined, fmt.Errorf("severity-number is out of range, the valid range is [1,24]")
	}

	// severity number should match well-known severityText
	switch severityText {
	case "Trace":
		if !(severityNumber >= 1 && severityNumber <= 4) {
			return "", 0, fmt.Errorf("severity text %q does not match severity number %d, the valid range is [1,4]", severityText, severityNumber)
		}
	case "Debug":
		if !(severityNumber >= 5 && severityNumber <= 8) {
			return "", 0, fmt.Errorf("severity text %q does not match severity number %d, the valid range is [5,8]", severityText, severityNumber)
		}
	case "Info":
		if !(severityNumber >= 9 && severityNumber <= 12) {
			return "", 0, fmt.Errorf("severity text %q does not match severity number %d, the valid range is [9,12]", severityText, severityNumber)
		}
	case "Warn":
		if !(severityNumber >= 13 && severityNumber <= 16) {
			return "", 0, fmt.Errorf("severity text %q does not match severity number %d, the valid range is [13,16]", severityText, severityNumber)
		}
	case "Error":
		if !(severityNumber >= 17 && severityNumber <= 20) {
			return "", 0, fmt.Errorf("severity text %q does not match severity number %d, the valid range is [17,20]", severityText, severityNumber)
		}
	case "Fatal":
		if !(severityNumber >= 21 && severityNumber <= 24) {
			return "", 0, fmt.Errorf("severity text %q does not match severity number %d, the valid range is [21,24]", severityText, severityNumber)
		}
	}

	return severityText, sn, nil
}
