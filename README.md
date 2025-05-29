# trazr-gen

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Go Reference](https://pkg.go.dev/badge/github.com/medxops/trazr-gen.svg)](https://pkg.go.dev/github.com/medxops/trazr-gen)
[![codecov](https://codecov.io/gh/<your-org>/<your-repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<your-org>/<your-repo>)

---

> **GPG Verification:** [Download the public GPG key](https://github.com/medxops/trazr-gen/raw/main/public.key) to verify signed releases and checksums.

---

**trazr-gen** is a modern, security-conscious tool for generating synthetic OpenTelemetry logs, metrics, and traces. It is designed for:
- Validating observability pipelines, masking, and alerting rules
- Testing OpenTelemetry Collector and backend configurations
- Simulating realistic workloads, including the injection of fake/test sensitive data (PII/PHI/credentials)
- Enabling security and compliance teams to audit and verify data handling in observability systems

trazr-gen is safe for use in any environment: **all sensitive data is fake/test only** and never real.

---

## Table of Contents
- [Security Note](#security-note)
- [Quick Start](#quick-start)
- [Supported Signals](#supported-signals)
- [Attribute-driven Data Injection](#attribute-driven-data-injection)
- [Trace Scenarios](#trace-scenarios)
- [Sensitive Attribute Keys](#sensitive-attribute-keys)
- [Loading Sensitive Data from a Config File](#loading-sensitive-data-from-a-config-file)
- [Observability & Security Best Practices](#observability--security-best-practices)
- [Getting Started](#getting-started)
- [Run](#run)
- [CLI Usage: Global vs. Subcommand Flags](#cli-usage-global-vs-subcommand-flags)
- [Configuration Parameters](#configuration-parameters)
- [Metrics-Specific Parameters](#metrics-specific-parameters)
- [Metrics Subcommands](#metrics-subcommands)
- [Getting Help](#getting-help)

---

## Security Note

🚨 **All sensitive data attributes used in this project are FAKE/TEST values.**
- No real PII, PHI, or secrets are present in the codebase.
- These values are for testing, validation, and observability pipeline development only.
- **Never use real customer or production data in synthetic generators or test code.**

---

# Quick Start

Generate a single trace with sensitive data:
```sh
trazr-gen traces single --output localhost:4317 --attributes sensitive
```

Generate multiple logs with sensitive data:
```sh
trazr-gen logs multi --output localhost:4317 --attributes sensitive
```

Override sensitive data with a YAML config:
```sh
trazr-gen traces single --output localhost:4317 --attributes sensitive --config path/to/trazrgen.yaml
```

---

## Supported Signals

- **Traces:** Simulate different trace patterns, span events, and links.
- **Metrics:** Exponential histogram, gauge, histogram, sum, exemplars.
- **Logs:** Log levels, log attributes, trace context correlation.

## Configuration Parameters

All parameters can be set via CLI flags, config file, or environment variables.

| Parameter            | Type                | Description                                                                                 | Default                        |
|----------------------|---------------------|---------------------------------------------------------------------------------------------|-------------------------------|
| `--service-name`     | `string`            | Name of the service to use in generated telemetry.                                           | `"trazr-gen"`                |
| `--duration`         | `time.Duration`     | Total duration to run the generator (e.g., `10s`, `1m`). Must be non-negative.              | `0` (unbounded if count is also 0) |
| `--output`           | `string`            | OTLP endpoint or output target (e.g., `localhost:4317`).                                    | `"terminal"`                 |
| `--insecure`         | `bool`              | Use insecure (non-TLS) connection for OTLP.                                                 | `false`                       |
| `--use-http`         | `bool`              | Use HTTP instead of gRPC for OTLP.                                                          | `false`                       |
| `--headers`          | `map[string]string` | Additional OTLP headers as key-value pairs.                                                 | `{}`                          |
| `--attributes`       | `[]string`          | List of attribute keys to inject (e.g., `sensitive`).                                       | `[]`                          |
| `--rate`             | `float64`           | Number of events (logs, traces, metrics) generated per second. 0 means unthrottled.         | `1`                           |

### Logs-Specific Parameters

| Parameter        | Type    | Description                                                        | Default |
|------------------|---------|--------------------------------------------------------------------|---------|
| `--num-logs`     | `int`   | Number of logs to generate. 0 = unbounded (unless --duration is set). | `0`     |

### Traces-Specific Parameters

| Parameter              | Type        | Description                                                        | Default                  |
|------------------------|-------------|--------------------------------------------------------------------|--------------------------|
| `--num-traces`         | `int`       | Number of traces to generate. 0 = unbounded (unless --duration is set). | `3` (multi), `1` (single) |
| `--propagate-context`  | `bool`      | Whether to propagate context between spans.                        | `false` (single), `false` (multi) |
| `--scenarios`          | `[]string`  | List of trace scenarios to run (e.g., `basic`).                    | `["basic"]`             |

### Metrics-Specific Parameters

All configuration for metrics is handled via common parameters.

## Metrics Subcommands

The `metrics` command supports the following subcommands, each simulating a different metric type:

| Subcommand                | Alias   | Description                                                      |
|---------------------------|---------|------------------------------------------------------------------|
| `gauge`                   | `g`     | Generate metrics of type gauge (values that can go up and down)   |
| `histogram`               | `hist`  | Generate metrics of type histogram (distribution of values)       |
| `exponential-histogram`   | `ehist` | Generate metrics of type exponential histogram (high dynamic range)|
| `sum`                     | `s`     | Generate metrics of type sum (additive values over time)          |

### Example Usage

```sh
trazr-gen metrics gauge --output terminal --min 0 --max 100 --unit "1" --temporality cumulative
trazr-gen metrics histogram --output terminal --bounds 1,5,10,25,50,100 --unit "ms"
trazr-gen metrics exponential-histogram --output terminal --scale 2 --max-size 1000
trazr-gen metrics sum --output terminal --monotonic true --unit "1"
```

### Subcommand-Specific Flags

- **gauge**
  - `--min` (float): Minimum value (default: 0)
  - `--max` (float): Maximum value (default: 100)
  - `--unit` (string): Unit of measurement (default: "1")
  - `--temporality` (string): "delta" or "cumulative" (default: "cumulative")

- **histogram**
  - `--bounds` (float list): Bucket boundaries (default: 1,5,10,25,50,100,250,500,1000)
  - `--unit` (string): Unit of measurement (default: "ms")
  - `--temporality` (string): "delta" or "cumulative" (default: "cumulative")
  - `--record-minmax` (bool): Record min/max values (default: true)

- **exponential-histogram**
  - `--scale` (int): Scale factor for buckets (default: 0)
  - `--max-size` (float): Maximum value to generate (default: 1000)
  - `--zero-threshold` (float): Threshold for zero bucket (default: 1e-6)
  - `--unit` (string): Unit of measurement (default: "ms")
  - `--temporality` (string): "delta" or "cumulative" (default: "cumulative")
  - `--record-minmax` (bool): Record min/max values (default: true)

- **sum**
  - `--monotonic` (bool): Whether the sum is monotonic (default: true)
  - `--unit` (string): Unit of measurement (default: "1")
  - `--temporality` (string): "delta" or "cumulative" (default: "cumulative")

### Temporality

The `--temporality` flag controls how metric values are aggregated and reported:

- **Accepted values:** `cumulative` (default), `delta`
- **Cumulative:** Reports the total value since the start of the process or instrument.
- **Delta:** Reports the change in value since the last export interval.

**Default:** If not specified, `cumulative` is used.

#### Supported and Recommended Usage
- For most use cases, `cumulative` is recommended and is the default.
- `delta` is supported for counters, updowncounters, and histograms, but may not be supported for all metric types or backends.
- For gauges, only `cumulative` is supported. If you select `delta` for a gauge, a warning will be logged and the tool may fall back to `cumulative`.

#### Example Usage

```sh
trazr-gen metrics gauge --temporality cumulative   # Default, recommended for most use cases
trazr-gen metrics sum --temporality delta         # Use delta, if supported by your backend
```

#### Warnings
- If you select `delta` temporality for a metric type or backend that does not support it, you will see a warning, and the tool may fall back to `cumulative`.
- Always check your backend's documentation for supported temporalities.

---

**Notes:**
- All durations are specified as Go duration strings (e.g., `10s`, `1m`).
- The `Output` parameter is required for all signals.
- The `Attributes` parameter can be used to inject special behaviors, such as fake sensitive data (`sensitive`).
- For a full list of CLI flags and their usage, run `trazr-gen --help` or see the help for each subcommand.

---

## Attribute-driven Data Injection

You can control special behaviors in trace and log generation using the `--attributes` flag. This allows you to inject sensitive (fake/test) data or other special patterns for pipeline validation and testing.

### Available Attributes
- `sensitive`: Randomly injects fake/test sensitive data (PII/PHI/credentials) into traces and logs, both as structured fields and sometimes in the log message body. More development is planned to augment the list of features for this parameter.

### Meta-Attributes for Sensitive Data

When sensitive data is injected (in traces or logs), the following two meta-attributes are automatically added:

- `mock.sensitive.present` (boolean): `true` if any sensitive data was injected into the record (span or log).
- `mock.sensitive.attributes` (string): a comma-separated list of the sensitive attribute keys injected (e.g., `user.ssn,user.email`).
  - If a sensitive attribute is injected into the log body, its key is included in this list as well.

These meta-attributes are present in both traces and logs, making it easy to audit, test, and filter for synthetic sensitive data in your observability pipeline.

### Usage Examples

> **Note:** The `--output` flag is required for all traces and logs commands.

Generate a single trace with sensitive data:
```sh
trazr-gen traces single --output localhost:4317 --attributes sensitive
```

Generate multiple logs with sensitive data:
```sh
trazr-gen logs multi --output localhost:4317 --attributes sensitive
```

You can combine attributes as more are added:
```sh
trazr-gen traces multi --output localhost:4317 --attributes sensitive,latency
```

Override sensitive data with a YAML config:
```sh
trazr-gen traces single --output localhost:4317 --attributes sensitive --config path/to/trazrgen.yaml
```

### What is injected?
When `--attributes sensitive` is used, the generator will randomly inject fake/test sensitive fields such as:
- SSN, email, phone, address, name, DOB
- Credit card, bank account
- Medical record number, diagnosis code, medication
- Auth tokens, IP addresses, URLs with PII/PHI
- Biometric and image data

**All values are fake/test and for validation only.**

---

## Trace Scenarios

The following trace scenarios are available:

| Scenario Name   | Description                                                                 | Sensitive Attribute Support |
|-----------------|-----------------------------------------------------------------------------|----------------------------|
| **basic**       | Simulates a simple client-server (ping-pong) interaction with two spans: a client "ping" and a server "pong". Supports injection of fake/test sensitive data if the `sensitive` attribute is set. | Yes                        |
| **web_mobile**  | Simulates a web or mobile client making a request to a backend, including web server, app server, and database spans. Models user agents, device types, and realistic HTTP/database attributes. | No                         |
| **eventing**    | Simulates an event-driven (producer/consumer) workflow, such as a Kafka or messaging system. Includes producer, consumer, and event processing spans, with realistic messaging attributes and links. | No                         |
| **microservices** | Simulates a complex, multi-service, multi-span workflow across a variety of microservices (API gateway, auth, user, product, payment, etc.). Each span represents an operation in a different service, with random attributes and occasional errors. | No                         |

**Notes:**
- The `basic` scenario is the only one that supports injection of fake/test sensitive data via the `sensitive` attribute.
- All other scenarios focus on realistic service-to-service, event-driven, or client-server traces, but do not inject sensitive data.

---

## Sensitive Attribute Keys
When using `--attributes sensitive`, the following fake/test sensitive fields may be injected:

- user.ssn
- user.email
- user.phone
- user.address
- user.dob
- user.name
- user.national_id
- passport.number
- driver_license.number
- credit.card
- bank.account
- health_plan.beneficiary_number
- medical_record.number
- health.diagnosis_code
- health.procedure_code
- health.medication
- device.serial_number
- db.statement
- url.full
- http.request.header.authorization
- http.request.header.x_patient_id
- net.peer.ip
- ip.address
- web.url
- biometric.fingerprint
- image.full_face

See `internal/attributes/sensitive.go` for the authoritative list and values.

---

## Loading Sensitive Data from a Config File

You can override the built-in sensitive data table by providing a YAML config file with the `--config` flag. This works for both traces and logs.

### Example YAML Config
```yaml
sensitive_data:
  - key: user.ssn
    value: "999-999-9999"
  - key: user.email
    value: "test@example.com"
  # ... more ...
```

### Usage

```sh
trazr-gen traces single --attributes sensitive --config path/to/trazrgen.yaml
trazr-gen logs multi --attributes sensitive --config path/to/trazrgen.yaml
```

- If provided, the config file **replaces** the built-in sensitive data table for the duration of the run.
- If not provided, the built-in (fake/test) sensitive data is used.

---

## Observability & Security Best Practices

- Use the `mock.sensitive.present` and `mock.sensitive.attributes` meta-attributes to:
  - Audit and validate masking or redaction in downstream systems.
  - Filter or alert on the presence of synthetic sensitive data in your observability pipeline.
  - Test data loss prevention (DLP), SIEM, or compliance tooling.
- Always keep the Security Note in mind: **never use real PII/PHI/secrets in test or synthetic data.**
- Document and review all custom sensitive data loaded via config files.

---

## Getting Started

Installing `trazr-gen` is possible via several methods. It can be installed via `brew`, a binary downloaded from GitHub [Releases](https://github.com/medxops/trazr-gen/releases), or running it as a distroless multi-arch docker image.

### brew

Install [brew](https://brew.sh/) and then run:

```sh
brew install medxops/tap/trazr-gen
```

### Download Binary

Download the latest version from the [Releases](https://github.com/medxops/trazr-gen/releases) page.

### Docker

To see all the tags view the [Packages](https://github.com/medxops/trazr-gen/pkgs/container/trazr-gen) page.

Run the container via the following command (remember to set --output):

```sh
docker run --rm ghcr.io/medxops/trazr-gen:latest traces single --output <your-output> vbutes sensitive
```

---

## Run

Running `trazr-gen` will generate this help:

```sh
NAME:
   trazr-gen - A tool to generate synthetic OpenTelemetry logs, metrics and traces

USAGE:
   trazr-gen command [command options] [arguments...]

VERSION:
   develop

COMMANDS:
   logs, l     Generate logs
   metrics, m  Generate metrics
   traces, t   Generate traces
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version


## Attribution

This project is a fork of [krzko/otelgen](https://github.com/krzko/otelgen) by [krzko](https://github.com/krzko).
The original project is licensed under the [Apache License 2.0](https://github.com/krzko/trazr-gen/blob/main/LICENSE).

## License

This project remains under the [Apache License 2.0](LICENSE).
If you make significant changes, you may add your own copyright notice for your contributions, but the original code and all derivatives must retain the Apache 2.0 license and attribution.

```

