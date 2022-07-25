# otelgen

A tool to generate synthetic OpenTelemetry logs, metrics and traces.

## Why

Often synthetics are used to validate  certain configurations, to ensure that that systems operate as expected. Operating [OpenTelemetry Collectors](https://opentelemetry.io/docs/collector/) is often a complex task, which entails tuning many facets such as [Receivers](https://opentelemetry.io/docs/collector/configuration/#receivers), [Proccessors](https://opentelemetry.io/docs/collector/configuration/#processors) and [Exporters](https://opentelemetry.io/docs/collector/configuration/#processors).

`otelgen` allows you to quickly validate these configurations using the [OpenTelemetry Protocol Specification](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md), which supports both [OTLP/gRPC](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md#otlpgrpc) and [OTLP/HTTP](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/otlp.md#otlphttp) using the [OTLP Receiver](https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver/otlpreceiver).

## Supported Specifications

The following specifications are supported.

- [ ] Logs: Not started yet
- [X] Metrics: Yes
  - Counter
  - Histogram
  - Gauge
  - UpDownCounter
- [X] Traces: Yes

## Getting Started

Installing `otelgen` is possible via several methods. It can be insatlled via `brew`, an binary downloaded from GitHub [Releases](https://github.com/krzko/otelgen/releases), or running it as a distroless multi-arch docker image.

### brew

Install [brew](https://brew.sh/) and then run:

```sh
brew install krzko/tap/otelgen
```

### Download Binary

Download the latest version from the [Releases](https://github.com/krzko/otelgen/releases) page.

### Docker

To see all the tags view the [Packages](https://github.com/krzko/proto2yaml/pkgs/container/proto2yaml) page.

Rn the container via the following command:

```sh
docker run --rm ghcr.io/krzko/otelgen:latest -h
```

## Run

Running `otelgen` will generate this help:

```sh
NAME:
   otelgen - A tool to generate synthetic OpenTelemetry logs, metrics and traces

USAGE:
   otelgen [global options] command [command options] [arguments...]

VERSION:
   v0.0.1

COMMANDS:
   metrics, m  Generate metrics
   traces, t   Generate traces
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --duration value, -d value           duration in seconds (default: 0)
   --help, -h                           show help (default: false)
   --insecure, -i                       whether to enable client transport security (default: false)
   --log-level value                    log level used by the logger, one of: debug, info, warn, error (default: "info")
   --otel-exporter-otlp-endpoint value  target URL to exporter endpoint
   --protocol value, -p value           the transport protocol, one of: grpc, http (default: "grpc")
   --rate value, -r value               rate in seconds (default: 5)
   --service-name value, -s value       service name to use (default: "otelgen")
   --version, -v                        print the version (default: false)
```

### Metrics

The `otelgen metrics` command supports many different **metric** types. Here is an example of how to generate metrics:

```sh
$ otelgen --otel-exporter-otlp-endpoint otelcol.foo.bar:443 metrics counter

{"level":"info","ts":1658746679.286242,"caller":"cli/metrics_counter.go:70","msg":"starting gRPC exporter"}
{"level":"info","ts":1658746679.46613,"caller":"cli/metrics_counter.go:87","msg":"Starting metrics generation"}
{"level":"info","ts":1658746679.466242,"caller":"metrics/config.go:59","msg":"generation of metrics is limited","per-second":5}
{"level":"info","ts":1658746679.467317,"caller":"metrics/metrics.go:47","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746684.4677298,"caller":"metrics/metrics.go:47","msg":"generating","name":"otelgen.metrics.counter"}
...
```

Here is an example, of how to limit the `duration` in seconds of a generation process:

```sh
$ otelgen --otel-exporter-otlp-endpoint otelcol.foo.bar:443 --duration 30 metrics counter

{"level":"info","ts":1658746721.598725,"caller":"cli/metrics_counter.go:70","msg":"starting gRPC exporter"}
{"level":"info","ts":1658746721.789262,"caller":"cli/metrics_counter.go:87","msg":"Starting metrics generation"}
{"level":"info","ts":1658746721.789321,"caller":"metrics/config.go:59","msg":"generation of metrics is limited","per-second":5}
{"level":"info","ts":1658746721.7894,"caller":"metrics/metrics.go:30","msg":"generation duration","seconds":30}
{"level":"info","ts":1658746721.789411,"caller":"metrics/metrics.go:40","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746726.7905679,"caller":"metrics/metrics.go:40","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746731.790965,"caller":"metrics/metrics.go:40","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746736.791102,"caller":"metrics/metrics.go:40","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746741.791389,"caller":"metrics/metrics.go:40","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746746.791574,"caller":"metrics/metrics.go:40","msg":"generating","name":"otelgen.metrics.counter"}
{"level":"info","ts":1658746751.791806,"caller":"cli/metrics_counter.go:79","msg":"stopping the exporter"}
```

### Traces

The `otelgen traces` command supports two types of traces, `single` and `multi`, the difference being, sometimes you just want to send a single trace to validate a configuration. **Multi** will allow you configure the `duration` and `rate`.

Here is an example, of how to generate a trace using `single`, with using secure transport:

```sh
$ otelgen --otel-exporter-otlp-endpoint otelcol.foo.bar:443 traces single

{"level":"info","ts":1658747062.525185,"caller":"cli/traces.go:90","msg":"starting gRPC exporter"}
{"level":"info","ts":1658747062.710507,"caller":"traces/config.go:58","msg":"generation of traces isn't being throttled"}
{"level":"info","ts":1658747062.7106712,"caller":"traces/traces.go:43","msg":"starting traces","worker":0}
{"level":"info","ts":1658747062.710735,"caller":"traces/traces.go:79","msg":"Trace","worker":0,"traceId":"9481f4c1a9099079c49ed14af2739b6d"}
{"level":"info","ts":1658747062.710753,"caller":"traces/traces.go:80","msg":"Parent Span","worker":0,"spanId":"fd76b9e4265aecfc"}
{"level":"info","ts":1658747062.7107708,"caller":"traces/traces.go:81","msg":"Child Span","worker":0,"spanId":"02267d8d1342d63a"}
{"level":"info","ts":1658747062.710814,"caller":"traces/traces.go:92","msg":"traces generated","worker":0,"traces":2}
{"level":"info","ts":1658747062.710835,"caller":"cli/traces.go:108","msg":"stop the batch span processor"}
{"level":"info","ts":1658747062.742642,"caller":"cli/traces.go:99","msg":"stopping the exporter"}
```

If you're running a collector on `localhost`, use `--insecure` to enable **h2c** for OTLP/gRPC (4317/tcp) and **http** for OTLP/HTTP (4318/tcp), of how to generate a trace using `single`, with using insecure transport:

```sh
$ otelgen --otel-exporter-otlp-endpoint localhost:4317 --insecure traces single
```

Here is an example, of how to generate a trace using `multi`, also, run `-h` to view the default values for each flag:

```sh
$ otelgen --otel-exporter-otlp-endpoint otelcol.foo.bar:443 --duration 10 --rate 1 traces multi

{"level":"info","ts":1658747148.7179039,"caller":"cli/traces.go:203","msg":"starting gRPC exporter"}
{"level":"info","ts":1658747148.908546,"caller":"traces/config.go:60","msg":"generation of traces is limited","per-second":1}
{"level":"info","ts":1658747148.908957,"caller":"traces/config.go:81","msg":"generation duration","seconds":10}
{"level":"info","ts":1658747148.910296,"caller":"traces/traces.go:43","msg":"starting traces","worker":0}
{"level":"info","ts":1658747148.91046,"caller":"traces/traces.go:79","msg":"Trace","worker":0,"traceId":"e299fc2461e04ee3c97d4f59e9b5f67a"}
{"level":"info","ts":1658747148.910481,"caller":"traces/traces.go:80","msg":"Parent Span","worker":0,"spanId":"0cefe413f4f5559a"}
{"level":"info","ts":1658747148.910497,"caller":"traces/traces.go:81","msg":"Child Span","worker":0,"spanId":"0ff83ff196aa83de"}
{"level":"info","ts":1658747148.91053,"caller":"traces/traces.go:43","msg":"starting traces","worker":0}
{"level":"info","ts":1658747149.9106922,"caller":"traces/traces.go:79","msg":"Trace","worker":0,"traceId":"9161121ffb377ef3e7b1d1efdb88c5d3"}
{"level":"info","ts":1658747149.910769,"caller":"traces/traces.go:80","msg":"Parent Span","worker":0,"spanId":"0aab1b9d6535bb84"}
{"level":"info","ts":1658747149.910798,"caller":"traces/traces.go:81","msg":"Child Span","worker":0,"spanId":"665b66edc4c7e26e"}
...
```

## Acknowledgements

This tool was developed in a short amount of time due to the awesome idea of the following sources:

- [tracegen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/tracegen): This utility simulates a client generating traces, useful for testing and demonstration purposes. In essence, `otelgen` uses `tracegen` as the tracing infrastructure.
- [Uptrace Metrics API](https://uptrace.dev/opentelemetry/go-metrics.html): Amazing metrics examples.
