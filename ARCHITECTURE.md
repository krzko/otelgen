# Project Architecture: trazr-gen

This document describes the high-level architecture, key packages, and design principles of the `trazr-gen` project.

---

## Overview

`trazr-gen` is a modular, idiomatic Go CLI application designed for observability, metrics, and trace data generation. The project follows Clean Architecture and modern Go best practices to ensure maintainability, testability, and scalability.

---

## Directory Structure

```
├── cmd/                # Application entrypoints (main.go for each CLI)
│   ├── trazr-gen/
│   └── testdata-gen/
├── internal/           # Core application logic (not exposed externally)
│   ├── metrics/        # Metrics generation logic (histograms, gauges, etc.)
│   ├── traces/         # Tracing scenario logic and helpers
│   ├── cli/            # CLI command definitions and argument parsing
│   ├── attributes/     # Attribute helpers for metrics/traces
│   └── logs/           # Logging utilities
├── build/, scripts/    # Build and utility scripts
├── .github/            # GitHub Actions workflows, issue/PR templates
├── Dockerfile, Makefile, .golangci.yml, etc.
```

---

## Key Packages and Responsibilities

- **cmd/**: Entrypoints for CLI applications. Each subdirectory contains a `main.go`.
- **internal/cli/**: CLI command definitions, argument parsing, and orchestration.
- **internal/metrics/**: Business logic for generating synthetic OpenTelemetry metrics (histogram, gauge, sum, etc.).
- **internal/traces/**: Business logic for generating synthetic traces and scenarios. Includes `scenarios/` for reusable trace patterns.
- **internal/attributes/**: Helpers for generating and injecting attributes into metrics and traces.
- **internal/logs/**: Logging utilities and configuration.

---

## Design Principles

- **Clean Architecture**: Separation of concerns between CLI, business logic, and data layers.
- **Domain-Driven Design**: Code grouped by feature/domain for clarity and cohesion.
- **Interface-Driven Development**: All public functions interact with interfaces, not concrete types, for testability and flexibility.
- **Dependency Injection**: Dependencies are injected via constructors, avoiding global state.
- **Observability**: OpenTelemetry is used for tracing, metrics, and logging. Context propagation is enforced throughout.
- **Testing**: Table-driven, parallelizable unit tests. Integration tests are separated by build tags.

---

## Observability & Instrumentation

- All metrics and traces are generated using OpenTelemetry APIs.
- Context (`context.Context`) is propagated through all major functions.
- Logging is structured and correlates with trace IDs.

---

## Extending the Project

- **Add a new CLI command**: Create a new file in `internal/cli/` and register it in the appropriate `main.go` under `cmd/`.
- **Add a new metric or trace scenario**: Implement in `internal/metrics/` or `internal/traces/scenarios/` and expose via CLI.
- **Add integration tests**: Place in a `test/` directory or use build tags for separation.

---

## References
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Clean Architecture](https://8thlight.com/blog/uncle-bob/2012/08/13/the-clean-architecture.html)
- [OpenTelemetry for Go](https://opentelemetry.io/docs/instrumentation/go/)

---

## Note on WorkerCount Parameter

The `WorkerCount` parameter exists in the configuration and codebase for future extensibility, but **is not currently implemented**. Regardless of the value set, only a single worker/goroutine is used for metric generation. This is a known architectural limitation and may be addressed in future versions.

## Note on NumMetrics Parameter

The `NumMetrics` parameter exists in the configuration and codebase for future extensibility, but **is not currently implemented**. The number of metrics generated is controlled by duration, not by this value. This is a known architectural limitation and may be addressed in future versions.

---

For more details, see the `README.md` or open an issue/discussion. 