//go:build !integration
// +build !integration

package cli

import (
	"context"
	"testing"

	"flag"

	"github.com/medxops/trazr-gen/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
)

func TestNewMetricExporter_UnsupportedProtocol(t *testing.T) {
	_, err := NewMetricExporter(context.TODO(), "notaproto", nil)
	if err == nil || err.Error() != "unsupported protocol: notaproto" {
		t.Errorf("expected unsupported protocol error, got %v", err)
	}
}

func TestParseAttributes_Valid(t *testing.T) {
	attrs := []string{"key1=val1", "key2=val2"}
	kvs, err := parseAttributes(attrs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(kvs) != 2 {
		t.Errorf("expected 2 attributes, got %d", len(kvs))
	}
}

func TestParseAttributes_InvalidFormat(t *testing.T) {
	attrs := []string{"key1val1"}
	_, err := parseAttributes(attrs)
	if err == nil {
		t.Error("expected error for invalid attribute format")
	}
}

func TestParseAttributes_EmptyKey(t *testing.T) {
	attrs := []string{"=val1"}
	_, err := parseAttributes(attrs)
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestPreferDeltaTemporalitySelector(t *testing.T) {
	cases := []struct {
		kind     metric.InstrumentKind
		expected metricdata.Temporality
	}{
		{metric.InstrumentKindCounter, metricdata.DeltaTemporality},
		{metric.InstrumentKindObservableCounter, metricdata.DeltaTemporality},
		{metric.InstrumentKindUpDownCounter, metricdata.DeltaTemporality},
		{metric.InstrumentKindHistogram, metricdata.DeltaTemporality},
		{metric.InstrumentKindObservableGauge, metricdata.CumulativeTemporality},
	}
	for _, c := range cases {
		if got := preferDeltaTemporalitySelector(c.kind); got != c.expected {
			t.Errorf("preferDeltaTemporalitySelector(%v) = %v, want %v", c.kind, got, c.expected)
		}
	}
}

func TestPreferCumulativeTemporalitySelector(t *testing.T) {
	cases := []struct {
		kind     metric.InstrumentKind
		expected metricdata.Temporality
	}{
		{metric.InstrumentKindCounter, metricdata.CumulativeTemporality},
		{metric.InstrumentKindObservableCounter, metricdata.CumulativeTemporality},
		{metric.InstrumentKindUpDownCounter, metricdata.CumulativeTemporality},
		{metric.InstrumentKindHistogram, metricdata.CumulativeTemporality},
		{metric.InstrumentKindObservableGauge, metricdata.DeltaTemporality},
	}
	for _, c := range cases {
		if got := preferCumulativeTemporalitySelector(c.kind); got != c.expected {
			t.Errorf("preferCumulativeTemporalitySelector(%v) = %v, want %v", c.kind, got, c.expected)
		}
	}
}

func TestParseAttributes(t *testing.T) {
	cases := []struct {
		name    string
		input   []string
		wantErr bool
	}{
		{"valid single", []string{"foo=bar"}, false},
		{"valid multiple", []string{"foo=bar", "baz=qux"}, false},
		{"invalid format", []string{"foobar"}, true},
		{"empty key", []string{"=bar"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseAttributes(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseAttributes(%v) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

// Mock cli.Context with StringSlice method

type fakeContext struct {
	headers []string
}

func (f *fakeContext) StringSlice(_ string) []string { return f.headers }

func TestParseHeaders(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "header"},
		},
	}
	// Valid header
	set := flag.NewFlagSet("test", 0)
	set.Var(cli.NewStringSlice("foo=bar", "baz=qux"), "header", "")
	ctx := cli.NewContext(app, set, nil)
	headers, err := parseHeaders(ctx)
	if err != nil {
		t.Errorf("expected no error for valid headers, got %v", err)
	}
	if len(headers) != 2 || headers["foo"] != "bar" || headers["baz"] != "qux" {
		t.Errorf("unexpected headers: %v", headers)
	}
	// Invalid header
	set = flag.NewFlagSet("test", 0)
	set.Var(cli.NewStringSlice("foobar"), "header", "")
	ctx = cli.NewContext(app, set, nil)
	_, err = parseHeaders(ctx)
	if err == nil {
		t.Error("expected error for invalid header format")
	}
}

func flagSetWithHeaders(headers []string) *flag.FlagSet {
	set := flag.NewFlagSet("test", 0)
	_ = set.Set("header", cli.NewStringSlice(headers...).String())
	return set
}

func TestConfigureLogging(t *testing.T) {
	// This test just ensures no panic and covers both branches
	app := &cli.App{Flags: []cli.Flag{&cli.StringFlag{Name: "log-level"}}}
	set := flag.NewFlagSet("test", 0)
	_ = set.Set("log-level", "debug")
	ctx := cli.NewContext(app, set, nil)
	configureLogging(ctx)

	set = flag.NewFlagSet("test", 0)
	_ = set.Set("log-level", "info")
	ctx = cli.NewContext(app, set, nil)
	configureLogging(ctx)
}

func TestGetExporterOptions(t *testing.T) {
	app := &cli.App{Flags: []cli.Flag{
		&cli.StringFlag{Name: "output"},
		&cli.BoolFlag{Name: "insecure"},
		&cli.StringFlag{Name: "temporality"},
		&cli.StringSliceFlag{Name: "header"},
	}}
	cases := []struct {
		name        string
		output      string
		insecure    bool
		temporality string
		headers     []string
		expectDelta bool
	}{
		{"default cumulative", "localhost:4317", false, "", nil, true},
		{"insecure", "localhost:4317", true, "", nil, true},
		{"delta temporality", "localhost:4317", false, "delta", nil, true},
		{"cumulative temporality", "localhost:4317", false, "cumulative", nil, false},
		{"with headers", "localhost:4317", false, "delta", []string{"foo=bar"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			set := flag.NewFlagSet("test", 0)
			_ = set.Set("output", tc.output)
			_ = set.Set("insecure", boolToString(tc.insecure))
			_ = set.Set("temporality", tc.temporality)
			if tc.headers != nil {
				_ = set.Set("header", cli.NewStringSlice(tc.headers...).String())
			}
			ctx := cli.NewContext(app, set, nil)
			mc := &metrics.Config{Output: tc.output}
			grpcOpts, httpOpts := getExporterOptions(ctx, mc)
			assert.NotNil(t, grpcOpts)
			assert.NotNil(t, httpOpts)
		})
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// Mocks for MetricExporter and zap.Logger for createExporter

type mockExporter struct{ metric.Exporter }

type mockLogger struct{ *zap.Logger }

func TestCreateExporter_TerminalAndStdout(t *testing.T) {
	app := &cli.App{Flags: []cli.Flag{
		&cli.StringFlag{Name: "output"},
		&cli.StringFlag{Name: "protocol"},
	}}
	for _, output := range []string{"terminal", "stdout"} {
		set := flag.NewFlagSet("test", 0)
		_ = set.Set("output", output)
		ctx := cli.NewContext(app, set, nil)
		exp, err := createExporter(context.Background(), ctx, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, exp)
	}
}

func TestCreateExporter_UnsupportedProtocol(t *testing.T) {
	app := &cli.App{Flags: []cli.Flag{
		&cli.StringFlag{Name: "output"},
		&cli.StringFlag{Name: "protocol"},
	}}
	set := flag.NewFlagSet("test", 0)
	_ = set.Set("output", "localhost:4317")
	_ = set.Set("protocol", "notaproto")
	ctx := cli.NewContext(app, set, nil)
	// Should fallback to grpc, not error
	exp, err := createExporter(context.Background(), ctx, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, exp)
}
