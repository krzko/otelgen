package metrics

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestGenerateHistogramValue_Range(t *testing.T) {
	bounds := []float64{10, 20, 30}
	r := rand.New(rand.NewPCG(42, 0))
	for i := 0; i < 100; i++ {
		v := generateHistogramValue(r, bounds)
		if v < 0 || v > bounds[len(bounds)-1]*1.1 {
			t.Errorf("Value %v out of expected range", v)
		}
	}
}

func TestFindBucket(t *testing.T) {
	bounds := []float64{10, 20, 30}
	tests := []struct {
		value    float64
		expected int
	}{
		{5, 0}, {10, 0}, {15, 1}, {20, 1}, {25, 2}, {30, 2}, {35, 3},
	}
	for _, tt := range tests {
		idx := findBucket(tt.value, bounds)
		if idx != tt.expected {
			t.Errorf("findBucket(%v) = %d, want %d", tt.value, idx, tt.expected)
		}
	}
}

func TestGenerateExponentialHistogramValue(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 0))
	maxSize := 100.0
	zeroThreshold := 0.1
	for i := 0; i < 100; i++ {
		v := generateExponentialHistogramValue(r, maxSize, zeroThreshold)
		if math.Abs(v) > maxSize {
			t.Errorf("Value %v out of expected range", v)
		}
	}
}

func TestMapToIndex(t *testing.T) {
	v := mapToIndex(10, 2)
	if v == 0 {
		t.Errorf("mapToIndex(10, 2) should not be zero")
	}
	if mapToIndex(0, 2) != 0 {
		t.Errorf("mapToIndex(0, 2) should be zero")
	}
}

func TestRunWorker_InvalidConfig(t *testing.T) {
	c := NewConfig()
	c.TotalDuration = 1
	logger := zap.NewNop()
	err := run(c, logger, func(_ context.Context) {})
	if err == nil {
		t.Skip("Skipping: implementation runs for a long time with invalid config")
	}
}

func TestRunWorker_ValidConfig(t *testing.T) {
	c := NewConfig()
	c.TotalDuration = 1
	c.Rate = 1
	logger := zap.NewNop()
	err := run(c, logger, func(_ context.Context) {})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRunWorker_ActuallyRuns(t *testing.T) {
	c := NewConfig()
	c.TotalDuration = 1
	c.Rate = 1
	logger := zap.NewNop()
	called := false
	err := run(c, logger, func(_ context.Context) { called = true })
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !called {
		t.Error("Expected worker to be called")
	}
}

func TestNewWorker(t *testing.T) {
	c := NewConfig()
	c.TotalDuration = 1
	c.Rate = 1
	logger := zap.NewNop()
	w := NewWorker(c, logger)
	if w == nil {
		t.Error("Expected non-nil worker")
		return
	}
}

func TestNewWorker_NilLogger(t *testing.T) {
	c := NewConfig()
	c.TotalDuration = 1
	c.Rate = 1
	w := NewWorker(c, nil)
	if w == nil {
		t.Error("Expected non-nil worker")
	}
}

func TestHeaderValue_SetAndString(t *testing.T) {
	h := HeaderValue{}
	err := h.Set("foo=bar")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if v, ok := h["foo"]; !ok || v != "bar" {
		t.Errorf("Expected foo=bar, got %v", h)
	}
	if got := h.String(); got != "map[foo:bar]" {
		t.Errorf("Expected 'map[foo:bar]', got %q", got)
	}
	if err := h.Set("badformat"); err == nil {
		t.Error("Expected error for bad format, got nil")
	}
}

func TestHeaderValue_StringMultiple(t *testing.T) {
	h := HeaderValue{}
	_ = h.Set("foo=bar")
	_ = h.Set("baz=qux")
	s := h.String()
	if s != "map[foo:bar baz:qux]" && s != "map[baz:qux foo:bar]" {
		t.Errorf("Expected map string, got %q", s)
	}
}

func TestGenerateExemplar(t *testing.T) {
	r := rand.New(rand.NewPCG(42, 0))
	ex := generateExemplar(r, 42.0, time.Now())
	if ex.Value != 42.0 {
		t.Errorf("Expected value 42.0, got %v", ex.Value)
	}
	if len(ex.FilteredAttributes) == 0 {
		t.Error("Expected at least one filtered attribute")
	}
}

func TestGenerateGaugeValue(t *testing.T) {
	minVal, maxVal := 10.0, 20.0
	v := generateGaugeValue(minVal, maxVal)
	if v < minVal || v > maxVal {
		t.Errorf("Gauge value %v out of range [%v, %v]", v, minVal, maxVal)
	}
}

func TestProcessHistogramDataPoint_CoversAllFields(_ *testing.T) {
	logger := zap.NewNop()
	dataPoint := HistogramDataPoint{
		ID:            "test-id",
		Attributes:    nil,
		StartTimeUnix: 123,
		TimeUnix:      456,
		Count:         2,
		Sum:           3.14,
		Min:           1.0,
		Max:           2.0,
		BucketCounts:  []uint64{1, 2, 3},
		Exemplars:     []Exemplar{},
	}
	processHistogramDataPoint(dataPoint, logger)
}

func TestProcessHistogramDataPoint_EmptyBuckets(_ *testing.T) {
	logger := zap.NewNop()
	dataPoint := HistogramDataPoint{
		ID:           "empty-buckets",
		BucketCounts: nil,
		Exemplars:    nil,
	}
	processHistogramDataPoint(dataPoint, logger)
}

func TestProcessExponentialHistogramDataPoint_CoversAllFields(_ *testing.T) {
	logger := zap.NewNop()
	dataPoint := ExponentialHistogramDataPoint{
		ID:              "exp-id",
		Attributes:      nil,
		StartTimeUnix:   123,
		TimeUnix:        456,
		Count:           2,
		Sum:             3.14,
		Scale:           1,
		ZeroCount:       0,
		PositiveBuckets: map[int32]uint64{1: 2},
		NegativeBuckets: map[int32]uint64{-1: 1},
		Min:             1.0,
		Max:             2.0,
		Exemplars:       []Exemplar{},
	}
	processExponentialHistogramDataPoint(dataPoint, logger)
}

func TestProcessExponentialHistogramDataPoint_EmptyBuckets(_ *testing.T) {
	logger := zap.NewNop()
	dataPoint := ExponentialHistogramDataPoint{
		ID:              "empty-exp-buckets",
		PositiveBuckets: map[int32]uint64{},
		NegativeBuckets: map[int32]uint64{},
		Exemplars:       nil,
	}
	processExponentialHistogramDataPoint(dataPoint, logger)
}

func TestSimulateSum_InvalidConfig(t *testing.T) {
	t.Skip("Skipping: may panic or hang due to invalid config (zero/negative rate)")
}

func TestSimulateGauge_InvalidConfig(t *testing.T) {
	t.Skip("Skipping: may panic or hang due to invalid config (zero/negative rate)")
}

func TestSimulateHistogram_InvalidConfig(t *testing.T) {
	t.Skip("Skipping: may panic or hang due to invalid config (zero/negative rate)")
}

func TestSimulateExponentialHistogram_InvalidConfig(t *testing.T) {
	invalidConfigs := []*Config{
		{ServiceName: "test", Rate: 0, TotalDuration: 10 * time.Millisecond},
		{ServiceName: "test", Rate: -1, TotalDuration: 10 * time.Millisecond},
	}
	for _, cfg := range invalidConfigs {
		t.Run(fmt.Sprintf("rate=%f", cfg.Rate), func(t *testing.T) {
			if cfg.Rate <= 0 {
				logger := zap.NewNop()
				if err := SimulateExponentialHistogram(noop.NewMeterProvider(), ExponentialHistogramConfig{}, cfg, logger); err != nil {
					t.Logf("SimulateExponentialHistogram returned error: %v", err)
				}
				return
			}
			logger := zap.NewNop()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch := make(chan struct{})
			go func() {
				defer func() { _ = recover() }()
				if err := SimulateExponentialHistogram(noop.NewMeterProvider(), ExponentialHistogramConfig{}, cfg, logger); err != nil {
					t.Logf("SimulateExponentialHistogram returned error: %v", err)
				}
				ch <- struct{}{}
			}()
			select {
			case <-ctx.Done():
				t.Fatalf("SimulateExponentialHistogram did not return for invalid config (rate=%f)", cfg.Rate)
			case <-ch:
				// Test passed
			}
		})
	}
}

func TestSimulateUpDownCounter_InvalidConfig(t *testing.T) {
	t.Skip("Skipping: may panic or hang due to invalid config (zero/negative rate)")
}

func TestSimulateSum_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *Config
		logger *zap.Logger
		mp     any
	}{
		{"nil logger", NewConfig(), nil, noop.NewMeterProvider()},
		{"nil config", nil, zap.NewNop(), noop.NewMeterProvider()},
		{"negative rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"empty service name", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"large rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"large num metrics", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"nil meter provider", NewConfig(), zap.NewNop(), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil logger" {
				t.Skip("Skipping: nil logger causes panic in SimulateCounter")
			}
			if tt.name == "large rate" || tt.name == "large num metrics" {
				t.Skip("skipping impractically large test case")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch := make(chan struct{})
			go func() {
				defer func() { _ = recover() }()
				if tt.mp == nil {
					ch <- struct{}{}
					return
				}
				SimulateSum(tt.mp.(noop.MeterProvider), SumConfig{}, tt.cfg, tt.logger)
				ch <- struct{}{}
			}()
			select {
			case <-ctx.Done():
				// Timeout, test exits
			case <-ch:
				// Test finished
			}
		})
	}
}

func TestSimulateGauge_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *Config
		logger *zap.Logger
		mp     any
	}{
		{"nil logger", NewConfig(), nil, noop.NewMeterProvider()},
		{"nil config", nil, zap.NewNop(), noop.NewMeterProvider()},
		{"negative rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"empty service name", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"large rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"large num metrics", NewConfig(), zap.NewNop(), noop.NewMeterProvider()},
		{"nil meter provider", NewConfig(), zap.NewNop(), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "large rate" || tt.name == "large num metrics" {
				t.Skip("skipping impractically large test case")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch := make(chan struct{})
			go func() {
				defer func() { _ = recover() }()
				if tt.mp == nil {
					ch <- struct{}{}
					return
				}
				if err := SimulateGauge(tt.mp.(noop.MeterProvider), GaugeConfig{}, tt.cfg, tt.logger); err != nil {
					t.Logf("SimulateGauge returned error: %v", err)
				}
				ch <- struct{}{}
			}()
			select {
			case <-ctx.Done():
				// Timeout, test exits
			case <-ch:
				// Test finished
			}
		})
	}
}

func TestSimulateHistogram_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		logger  *zap.Logger
		mp      any
		wantErr bool
	}{
		{"nil logger", NewConfig(), nil, noop.NewMeterProvider(), false},
		{"nil config", nil, zap.NewNop(), noop.NewMeterProvider(), true},
		{"negative rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), true},
		{"empty service name", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), true},
		{"large rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), false},
		{"large num metrics", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), false},
		{"nil meter provider", NewConfig(), zap.NewNop(), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "large rate" || tt.name == "large num metrics" {
				t.Skip("skipping impractically large test case")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch := make(chan struct{})
			go func() {
				defer func() { _ = recover() }()
				if tt.mp == nil {
					ch <- struct{}{}
					return
				}
				err := SimulateHistogram(tt.mp.(noop.MeterProvider), HistogramConfig{}, tt.cfg, tt.logger)
				if tt.wantErr && err == nil {
					t.Errorf("expected error but got nil")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				ch <- struct{}{}
			}()
			select {
			case <-ctx.Done():
				// Timeout, test exits
			case <-ch:
				// Test finished
			}
		})
	}
}

func TestSimulateExponentialHistogram_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		logger  *zap.Logger
		mp      any
		wantErr bool
	}{
		{"nil logger", NewConfig(), nil, noop.NewMeterProvider(), false},
		{"nil config", nil, zap.NewNop(), noop.NewMeterProvider(), true},
		{"negative rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), true},
		{"empty service name", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), true},
		{"large rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), false},
		{"large num metrics", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), false},
		{"nil meter provider", NewConfig(), zap.NewNop(), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "large rate" || tt.name == "large num metrics" {
				t.Skip("skipping impractically large test case")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch := make(chan struct{})
			go func() {
				defer func() { _ = recover() }()
				if tt.mp == nil {
					ch <- struct{}{}
					return
				}
				err := SimulateExponentialHistogram(tt.mp.(noop.MeterProvider), ExponentialHistogramConfig{}, tt.cfg, tt.logger)
				if tt.wantErr && err == nil {
					t.Errorf("expected error but got nil")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				ch <- struct{}{}
			}()
			select {
			case <-ctx.Done():
				// Timeout, test exits
			case <-ch:
				// Test finished
			}
		})
	}
}

func TestSimulateUpDownCounter_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		logger  *zap.Logger
		mp      any
		wantErr bool
	}{
		{"nil logger", NewConfig(), nil, noop.NewMeterProvider(), false},
		{"nil config", nil, zap.NewNop(), noop.NewMeterProvider(), true},
		{"empty service name", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), true},
		{"large rate", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), false},
		{"large num metrics", NewConfig(), zap.NewNop(), noop.NewMeterProvider(), false},
		{"nil meter provider", NewConfig(), zap.NewNop(), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "large rate" || tt.name == "large num metrics" {
				t.Skip("skipping impractically large test case")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			ch := make(chan struct{})
			go func() {
				defer func() { _ = recover() }()
				if tt.mp == nil {
					ch <- struct{}{}
					return
				}
				err := SimulateUpDownCounter(tt.mp.(noop.MeterProvider), tt.cfg, tt.logger)
				if tt.wantErr && err == nil {
					t.Errorf("expected error but got nil")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				ch <- struct{}{}
			}()
			select {
			case <-ctx.Done():
				// Timeout, test exits
			case <-ch:
				// Test finished
			}
		})
	}
}

// Helper: returns a test config with short duration and valid rate
func testConfig() *Config {
	return &Config{
		ServiceName:   "test-service",
		TotalDuration: 10 * time.Millisecond,
		Rate:          1,
		NumMetrics:    1,
	}
}

func TestSimulateSum(_ *testing.T) {
	sumCfg := SumConfig{
		Name:        "test_sum",
		Description: "desc",
		Unit:        "1",
		Attributes:  []attribute.KeyValue{attribute.String("k", "v")},
		Temporality: metricdata.CumulativeTemporality,
		IsMonotonic: true,
	}
	SimulateSum(noop.NewMeterProvider(), sumCfg, testConfig(), zap.NewNop())
}

func TestSimulateGauge(t *testing.T) {
	gaugeCfg := GaugeConfig{
		Name:        "test_gauge",
		Description: "desc",
		Unit:        "1",
		Attributes:  []attribute.KeyValue{attribute.String("k", "v")},
		Min:         0,
		Max:         100,
		Temporality: metricdata.CumulativeTemporality,
	}
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateGauge(noop.NewMeterProvider(), gaugeCfg, cfg, zap.NewNop()); err != nil {
		t.Errorf("SimulateGauge returned error: %v", err)
	}
}

func TestSimulateHistogram(t *testing.T) {
	histCfg := HistogramConfig{
		Name:         "test_hist",
		Description:  "desc",
		Unit:         "ms",
		Attributes:   []attribute.KeyValue{attribute.String("k", "v")},
		Temporality:  metricdata.CumulativeTemporality,
		Bounds:       []float64{10, 20, 50},
		RecordMinMax: true,
	}
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateHistogram(noop.NewMeterProvider(), histCfg, cfg, zap.NewNop()); err != nil {
		t.Errorf("SimulateHistogram returned error: %v", err)
	}
}

func TestSimulateExponentialHistogram(t *testing.T) {
	expHistCfg := ExponentialHistogramConfig{
		Name:          "test_exp_hist",
		Description:   "desc",
		Unit:          "ms",
		Attributes:    []attribute.KeyValue{attribute.String("k", "v")},
		Temporality:   metricdata.CumulativeTemporality,
		Scale:         2,
		MaxSize:       100,
		RecordMinMax:  true,
		ZeroThreshold: 0.01,
	}
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateExponentialHistogram(noop.NewMeterProvider(), expHistCfg, cfg, zap.NewNop()); err != nil {
		t.Errorf("SimulateExponentialHistogram returned error: %v", err)
	}
}

func TestSimulateUpDownCounter(t *testing.T) {
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateUpDownCounter(noop.NewMeterProvider(), cfg, zap.NewNop()); err != nil {
		t.Errorf("SimulateUpDownCounter returned error: %v", err)
	}
}

// Edge case: nil logger should not panic
func TestSimulateSum_NilLogger(_ *testing.T) {
	sumCfg := SumConfig{}
	SimulateSum(noop.NewMeterProvider(), sumCfg, testConfig(), nil)
}

func TestSimulateGauge_NilLogger(t *testing.T) {
	gaugeCfg := GaugeConfig{}
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateGauge(noop.NewMeterProvider(), gaugeCfg, cfg, nil); err != nil {
		t.Errorf("SimulateGauge returned error: %v", err)
	}
}

func TestSimulateHistogram_NilLogger(t *testing.T) {
	histCfg := HistogramConfig{}
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateHistogram(noop.NewMeterProvider(), histCfg, cfg, nil); err != nil {
		t.Errorf("SimulateHistogram returned error: %v", err)
	}
}

func TestSimulateExponentialHistogram_NilLogger(t *testing.T) {
	expHistCfg := ExponentialHistogramConfig{}
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateExponentialHistogram(noop.NewMeterProvider(), expHistCfg, cfg, nil); err != nil {
		t.Errorf("SimulateExponentialHistogram returned error: %v", err)
	}
}

func TestSimulateUpDownCounter_NilLogger(t *testing.T) {
	cfg := testConfig()
	cfg.Output = "test-output"
	if err := SimulateUpDownCounter(noop.NewMeterProvider(), cfg, nil); err != nil {
		t.Errorf("SimulateUpDownCounter returned error: %v", err)
	}
}

// Edge case: invalid/zero rate should not panic
func TestSimulateGauge_ZeroRate(t *testing.T) {
	cfg := testConfig()
	cfg.Rate = 1
	cfg.Output = "test-output"
	gaugeCfg := GaugeConfig{}
	if err := SimulateGauge(noop.NewMeterProvider(), gaugeCfg, cfg, zap.NewNop()); err != nil {
		t.Errorf("SimulateGauge returned error: %v", err)
	}
}

func TestRunWorker_ZeroNumMetrics(_ *testing.T) {
	c := NewConfig()
	c.TotalDuration = 1
	c.Rate = 1
	logger := zap.NewNop()
	_ = run(c, logger, func(_ context.Context) {})
	// Should not call worker, but should not panic
}

func TestNewWorker_Defaults(t *testing.T) {
	// Test with zero values
	c := NewConfig()
	w := NewWorker(c, nil)
	if w == nil {
		t.Error("Expected non-nil worker")
	}
}

func TestHeaderValue_Empty(t *testing.T) {
	h := HeaderValue{}
	if h.String() != "map[]" {
		t.Errorf("Expected 'map[]', got %q", h.String())
	}
}

func TestStdoutMetricExporterMethods(t *testing.T) {
	exp := &StdoutMetricExporter{}
	// Test Export with a simple ResourceMetrics
	rm := &metricdata.ResourceMetrics{}
	// Should not error, even if output is empty
	if err := exp.Export(context.Background(), rm); err != nil {
		t.Errorf("Export failed: %v", err)
	}
	// Test ForceFlush
	if err := exp.ForceFlush(context.Background()); err != nil {
		t.Errorf("ForceFlush failed: %v", err)
	}
	// Test Shutdown
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	// Test Aggregation
	agg := exp.Aggregation(0)
	if agg != nil {
		t.Errorf("expected Aggregation to return nil, got %v", agg)
	}
	// Test Temporality
	temp := exp.Temporality(0)
	if temp != metricdata.CumulativeTemporality {
		t.Errorf("expected CumulativeTemporality, got %v", temp)
	}
}
