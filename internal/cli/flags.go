package cli

import (
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func getGlobalFlags() []cli.Flag {
	return []cli.Flag{
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "duration",
			Aliases: []string{"d"},
			Usage:   "duration in seconds",
			Value:   0,
		}),
		altsrc.NewStringSliceFlag(&cli.StringSliceFlag{
			Name: "header",
			// Aliases: []string{"h"},
			Usage: "additional headers in 'key=value' format",
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:    "insecure",
			Usage:   "whether to enable client transport security",
			Aliases: []string{"i"},
			// EnvVars: []string{"OTEL_EXPORTER_OTLP_INSECURE"},
			Value: false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "log-level",
			Usage: "log level used by the logger, one of: debug, info, warn, error",
			// EnvVars: []string{"OTEL_LOG_LEVEL"},
			Value: "info",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "otel-exporter-otlp-endpoint",
			Usage: "target URL to exporter endpoint",
			// EnvVars: []string{"OTEL_EXPORTER_OTLP_ENDPOINT"},
			// Required: true,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "protocol",
			Usage:   "the transport protocol, one of: grpc, http",
			Aliases: []string{"p"},
			// EnvVars: []string{"OTEL_EXPORTER_OTLP_PROTOCOL"},
			Value: "grpc",
		}),
		altsrc.NewInt64Flag(&cli.Int64Flag{
			Name:    "rate",
			Aliases: []string{"r"},
			Usage:   "rate in seconds",
			Value:   5,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "service-name",
			Usage:   "service name to use",
			Aliases: []string{"s"},
			// EnvVars: []string{"OTEL_SERVICE_NAME"},
			Value: "otelgen",
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "delta-temporality",
			Usage: "Use delta temporality when exporting metrics (cumulative temporality is the default)",
			Value: false,
		}),
	}
}
