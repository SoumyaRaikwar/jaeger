# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

Jaeger is a distributed tracing platform built on top of the OpenTelemetry Collector. Version 2 is a complete rewrite that leverages OTel Collector as its core, extending it with Jaeger-specific components (extensions, exporters, receivers).

## Development Commands

### Setup & Prerequisites
```bash
# Initialize submodules (required for jaeger-ui and idl)
git submodule update --init --recursive

# Install development tools (golangci-lint, gofumpt, mockery, etc.)
make install-tools
```

### Building
```bash
# Build the main Jaeger v2 binary
make build-jaeger

# Build all binaries for current platform (jaeger, remote-storage, examples, utilities)
make build-binaries

# Build for specific platforms
make build-binaries-linux-amd64
make build-binaries-darwin-arm64

# Build with UI (compiles React UI from jaeger-ui submodule)
make build-ui

# Run all-in-one locally (includes UI build)
make run-all-in-one
```

### Testing
```bash
# Run all unit tests (includes memory_storage_integration tag)
make test

# Run tests with coverage report
make cover

# Run integration tests for Jaeger v2 storage backends
# STORAGE can be: badger, cassandra, grpc, kafka, memory_v2, query
STORAGE=badger make jaeger-v2-storage-integration-test

# Run legacy storage integration tests
STORAGE=badger make storage-integration-test

# Run all-in-one integration test
make all-in-one-integration-test

# Run tail sampling integration test
make tail-sampling-integration-test
```

### Code Quality
```bash
# Auto-format code (gofumpt, import ordering, license headers)
make fmt

# Run all linters
make lint

# Individual lint checks
make lint-fmt          # Check formatting
make lint-license      # Check license headers
make lint-imports      # Check import ordering
make lint-go          # Run golangci-lint
make lint-goleak      # Check for goroutine leaks in tests
```

### Running Individual Tests
```bash
# Run specific test package
go test ./internal/storage/badger/...

# Run specific test function
go test ./cmd/jaeger/internal/integration -run TestBadgerStorage -v

# Run with race detector (automatically enabled on most platforms)
go test -race ./...

# Run test with storage integration tag
go test -tags=memory_storage_integration ./internal/storage/integration/...
```

## Architecture

### High-Level Structure

Jaeger v2 is built as an **OpenTelemetry Collector distribution** with custom Jaeger components:

- **Extensions**: `jaeger_storage`, `jaeger_query`, `remote_sampling`, `storage_cleaner` (testing only)
- **Receivers**: Standard OTLP + Jaeger/Zipkin legacy formats + Kafka
- **Exporters**: `jaeger_storage_exporter` (writes to storage backends), Kafka, Prometheus
- **Processors**: Adaptive sampling, tail sampling, batch processing

### Storage Architecture

**V2 Storage Interface** (`internal/storage/v2/`): New interface aligned with OTel semantics
- Backends: Memory, Badger, Cassandra, Elasticsearch, ClickHouse, gRPC remote storage
- Storage is accessed via `jaegerstorage` extension which provides factory pattern for multiple named backends
- Configuration is YAML-based (OpenTelemetry Collector format)

**V1 Storage Interface** (`internal/storage/v1/`): Legacy interface for backward compatibility
- Still used by some components during v1→v2 migration

### Key Extension: jaeger_storage

The `jaeger_storage` extension (`cmd/jaeger/internal/extension/jaegerstorage/`) is central to Jaeger's architecture:
- Manages multiple named storage backends (e.g., "some_store", "another_store")
- Provides storage factories to exporters and query extension
- Supports both primary and archive storage

### Key Extension: jaeger_query

The `jaeger_query` extension (`cmd/jaeger/internal/extension/jaegerquery/`) provides:
- gRPC query API for reading traces
- HTTP API serving Jaeger UI
- Span reader interface to storage backends

### Integration Testing Architecture

Integration tests in `cmd/jaeger/internal/integration/` are **E2E tests** that:
1. Start the full Jaeger v2 binary as a subprocess
2. Write spans via OTLP receiver (gRPC/HTTP)
3. Read spans via jaeger_query extension gRPC API
4. Clean storage between tests via `storage_cleaner` extension's `/purge` HTTP endpoint

This differs from unit-mode tests in `internal/storage/integration/` which test storage APIs directly.

## Project Structure

```
cmd/
  jaeger/                 # Main Jaeger v2 binary (OpenTelemetry Collector based)
    internal/
      extension/          # Jaeger extensions: jaeger_storage, jaeger_query, etc.
      exporters/          # Custom exporters (storageexporter)
      processors/         # Custom processors (adaptivesampling)
      integration/        # E2E integration tests
    config-*.yaml         # Example configs for different backends
  all-in-one/            # Legacy all-in-one (being phased out in favor of jaeger)
  remote-storage/        # Shared storage server (exposes single storage via gRPC)
  query/                 # Legacy query service
  collector/             # Legacy collector
  [utilities...]         # tracegen, anonymizer, es-index-cleaner, etc.

internal/
  storage/
    v2/                  # New storage interface (aligned with OTel)
    v1/                  # Legacy storage interface
  config/                # Shared configuration utilities

jaeger-ui/               # Submodule - React frontend
idl/                     # Submodule - Protobuf/Thrift data models
```

## Configuration

Jaeger v2 uses **OpenTelemetry Collector YAML configuration**. See `cmd/jaeger/config-*.yaml` for examples.

Key configuration sections:
- `extensions.jaeger_storage`: Define storage backends
- `extensions.jaeger_query`: Configure query service and UI
- `receivers.otlp`: Configure span ingestion endpoints
- `exporters.jaeger_storage_exporter`: Route traces to storage
- `service.pipelines`: Wire receivers → processors → exporters

## Code Conventions

### Import Ordering (enforced by `make fmt`)
```go
import (
    // 1. Standard library
    "fmt"
    "context"

    // 2. External dependencies
    "go.uber.org/zap"
    "go.opentelemetry.io/collector/component"

    // 3. Jaeger imports
    "github.com/jaegertracing/jaeger/internal/storage"
)
```

### Testing
- All packages must have at least one `*_test.go` file (coverage requirement: 95%)
- Use `.nocover` file to exclude packages that need external dependencies
- All test packages should include goleak check in `TestMain`
- Feature gates: Use OTel's feature gate system for managing breaking changes

### Code Formatting
- Use `gofumpt` (stricter than gofmt)
- Run `make fmt` before committing - it handles formatting, imports, and license headers

## Important Notes

- **Submodules**: `jaeger-ui` and `idl` are Git submodules. Changes there require separate PRs to their repositories.
- **Auto-generated files**: Never edit `*.pb.go`, `*_mock.go`, or files in `internal/proto-gen/`
- **DCO**: All commits must be signed (Developer Certificate of Origin)
- **Makefile**: Most development workflows go through Make targets, not direct Go commands
- **Version compatibility**: Jaeger maintains 3-month grace period for deprecated configuration options
- **Go versions**: Tracks currently supported Go versions; removing old Go support is not breaking

## Running with Different Storage Backends

```bash
# Memory (default for testing)
./cmd/jaeger/jaeger --config cmd/jaeger/config-badger.yaml

# Badger (embedded DB)
./cmd/jaeger/jaeger --config cmd/jaeger/config-badger.yaml

# Elasticsearch
./cmd/jaeger/jaeger --config cmd/jaeger/config-elasticsearch.yaml

# Cassandra
./cmd/jaeger/jaeger --config cmd/jaeger/config-cassandra.yaml

# Remote storage (gRPC)
./cmd/jaeger/jaeger --config cmd/jaeger/config-remote-storage.yaml
```

## Debugging Integration Tests

Integration tests can fail if storage isn't cleaned properly or if ports conflict. Check:
- Storage cleaner extension is configured (`storage_cleaner` in extensions)
- Ports are available (default: 4317 gRPC, 4318 HTTP, 16685 query, 8888 metrics)
- Previous test processes are terminated
