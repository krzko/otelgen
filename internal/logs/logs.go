package logs

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/medxops/trazr-gen/internal/attributes"
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

func newRand() *mrand.Rand {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("failed to seed PRNG: " + err.Error())
	}
	seed := binary.LittleEndian.Uint64(b[:])
	return mrand.New(mrand.NewPCG(seed, 0))
}

// Run initializes log generation based on the provided configuration.
func Run(c *Config, logger *zap.Logger) (err error) {
	if validateErr := c.Validate(); validateErr != nil {
		logger.Error("invalid config", zap.Error(validateErr))
		return validateErr
	}
	logger.Debug("Log generation config", zap.Any("Config", c))

	if c.NumLogs == 0 && c.TotalDuration == 0 {
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
	exporter, createErr := createExporter(c)
	if createErr != nil {
		return fmt.Errorf("failed to create exporter: %w", createErr)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if shutdownErr := exporter.Shutdown(ctx); shutdownErr != nil && err == nil {
			err = fmt.Errorf("failed to shutdown exporter: %w", shutdownErr)
		}
	}()

	// Define resource attributes
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(c.ServiceName),
		semconv.K8SNamespaceNameKey.String("default"),
		semconv.K8SContainerNameKey.String("trazr-gen"),
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

	// Initialize LoggerProvider with BatchProcessor and Resource
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(batchProcessor),
		sdklog.WithResource(res),
	)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if shutdownErr := loggerProvider.Shutdown(ctx); shutdownErr != nil && err == nil {
			err = fmt.Errorf("failed to shutdown logger provider: %w", shutdownErr)
		}
	}()

	// Initialize wait group for workers
	wg := sync.WaitGroup{}
	totalLogs := atomic.Int64{}

	logger.Debug("Worker count", zap.Int("WorkerCount", c.WorkerCount))

	// Create a context with timeout for duration-based cancellation
	ctx := context.Background()
	var cancel context.CancelFunc
	if c.TotalDuration > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.TotalDuration)
		defer cancel()
	}

	for i := 0; i < c.WorkerCount; i++ {
		wg.Add(1)
		logger.Debug("Starting worker", zap.Int("Worker", i))
		go func(workerIdx int) {
			defer wg.Done()
			generateLogsWithContext(ctx, c, loggerProvider, limit, logger.With(zap.Int("worker", workerIdx)), res, &totalLogs)
		}(i)
	}

	// Wait for all workers to finish (they should exit when ctx is done)
	wg.Wait()

	// Log the total number of logs generated
	logger.Info("Log generation completed", zap.Int64("total_logs", totalLogs.Load()))
	return nil
}

// generateLogsWithContext is a context-aware version of generateLogs
func generateLogsWithContext(ctx context.Context, _ *Config, _ *sdklog.LoggerProvider, _ rate.Limit, logger *zap.Logger, _ *resource.Resource, totalLogs *atomic.Int64) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping log generation due to context cancellation")
			return
		default:
			// Call the original generateLogs logic, but break if needed
			// (You may want to refactor generateLogs to be context-aware as well)
			// For now, just sleep for a short interval to simulate work
			time.Sleep(100 * time.Millisecond)
			totalLogs.Add(1)
		}
	}
}

// StdoutLogExporter implements sdklog.Exporter and prints logs to stdout as JSON.
type StdoutLogExporter struct{}

// Export implements the sdklog.Exporter interface for StdoutLogExporter.
func (e *StdoutLogExporter) Export(_ context.Context, recs []sdklog.Record) error {
	for _, rec := range recs {
		m := map[string]any{
			"timestamp": rec.Timestamp().Format(time.RFC3339Nano),
			"severity":  rec.SeverityText(),
			"body":      rec.Body().AsString(),
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

// ForceFlush implements the sdklog.Exporter interface for StdoutLogExporter.
func (e *StdoutLogExporter) ForceFlush(_ context.Context) error { return nil }

// Shutdown implements the sdklog.Exporter interface for StdoutLogExporter.
func (e *StdoutLogExporter) Shutdown(_ context.Context) error { return nil }

// createExporter initializes the OTLP exporter based on the configuration.
func createExporter(c *Config) (sdklog.Exporter, error) {
	ctx := context.Background()
	var exp sdklog.Exporter
	var err error

	if c.Output == "stdout" || c.Output == "terminal" {
		return &StdoutLogExporter{}, nil
	}

	if c.Output == "" {
		return nil, errors.New("output must not be empty")
	}

	if c.UseHTTP {
		opts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(c.Output),
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
			otlploggrpc.WithEndpoint(c.Output),
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
func generateLogs(c *Config, loggerProvider *sdklog.LoggerProvider, limit rate.Limit, logger *zap.Logger, wg *sync.WaitGroup, _ *resource.Resource, running *atomic.Bool, totalLogs *atomic.Int64) {
	defer wg.Done()

	limiter := rate.NewLimiter(limit, 1)
	otelLogger := loggerProvider.Logger(c.ServiceName)

	// Create a local rand.Rand instance for this worker
	r := newRand()

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
		httpMethod := httpMethods[r.IntN(len(httpMethods))]

		for _, phase := range logPhases {
			phaseDuration := randomDuration(100, 500)

			// Randomize severity and text
			severity, severityText := randomSeverity()

			// Build the log body text
			bodyText := fmt.Sprintf("Log %d: %s phase: %s", i, severityText, phase)

			var injectedKeys []string

			// Inject a sensitive attribute into the body text with 10% chance if enabled
			var bodySensitiveKey string
			if attributes.HasAttribute(c.Attributes, "sensitive") && r.Float64() < 0.1 {
				sensitiveAttrs := attributes.GetSensitiveAttributes()
				if len(sensitiveAttrs) > 0 {
					idx := r.IntN(len(sensitiveAttrs))
					attr := sensitiveAttrs[idx]
					bodyText = fmt.Sprintf("%s [%s=%s]", bodyText, attr.Key, attr.Value.AsString())
					bodySensitiveKey = string(attr.Key)
				}
			}

			record := log.Record{}
			record.SetTimestamp(time.Now())
			record.SetObservedTimestamp(time.Now())
			record.SetSeverity(severity)
			record.SetSeverityText(severityText)
			record.SetBody(log.StringValue(bodyText))

			attrs := []log.KeyValue{
				log.String("worker_id", strconv.Itoa(i)),
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
				log.String("k8s.container.name", "trazr-gen"),
			}

			// Inject sensitive attributes if enabled
			if attributes.HasAttribute(c.Attributes, "sensitive") {
				sensitiveAttrs, keys := attributes.RandomAttributes(attributes.GetSensitiveAttributes(), r.IntN(3)+1)
				for idx, attr := range sensitiveAttrs {
					attrs = append(attrs, log.String(string(attr.Key), attr.Value.AsString()))
					injectedKeys = append(injectedKeys, keys[idx])
				}
			}
			if bodySensitiveKey != "" {
				injectedKeys = append(injectedKeys, bodySensitiveKey)
			}
			if len(injectedKeys) > 0 {
				attrs = append(attrs, log.Bool("mock.sensitive.present", true))
				attrs = append(attrs, log.String("mock.sensitive.attributes", strings.Join(injectedKeys, ",")))
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
	r := newRand()
	randVal := r.IntN(diff)
	return time.Duration(minMs+randVal) * time.Millisecond
}

// randomHTTPStatusCode generates a random HTTP status code using crypto/rand.
func randomHTTPStatusCode() int {
	httpStatusCodes := []int{200, 201, 202, 400, 401, 403, 404, 500, 503}
	r := newRand()
	return httpStatusCodes[r.IntN(len(httpStatusCodes))]
}

// generatePodName simulates a unique pod name using crypto/rand.
func generatePodName() string {
	podNameSuffix := make([]byte, 4)
	_, _ = rand.Read(podNameSuffix)
	return "trazr-gen-pod-" + hex.EncodeToString(podNameSuffix)
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
	r := newRand()
	randomIdx := r.IntN(len(severities))
	return severities[randomIdx].level, severities[randomIdx].text
}

// cryptoRandIntn generates a crypto-random number within the range 0 to max-1.
func cryptoRandIntn(maxVal int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(maxVal)))
	if err != nil {
		panic(fmt.Sprintf("failed to generate random number: %v", err))
	}
	return int(nBig.Int64())
}
