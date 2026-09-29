# Macro

[![Tests](https://github.com/railzwaylabs/macro/actions/workflows/tests.yml/badge.svg)](https://github.com/railzwaylabs/macro/actions/workflows/tests.yml)
[![Lint](https://github.com/railzwaylabs/macro/actions/workflows/lint.yml/badge.svg)](https://github.com/railzwaylabs/macro/actions/workflows/lint.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/railzwaylabs/macro.svg)](https://pkg.go.dev/github.com/railzwaylabs/macro)
[![Release](https://img.shields.io/github/v/release/railzwaylabs/macro?sort=semver)](https://github.com/railzwaylabs/macro/releases)

Macro is a lightweight Go service toolkit and CLI. It provides a small runtime
at the executable boundary and scaffolds consistent services, workers, jobs,
and multi-project workspaces without taking ownership of application logic.

Macro is inspired by [Go Micro](https://github.com/micro/go-micro) and adapted
to the conventions used by Railzway services.

> **Project status:** Macro is under active development. The first development
> release is `v0.1.0`, and APIs may evolve before the first stable release.

## Why Macro?

Go workloads often repeat the same setup: process lifecycle, signal handling,
logging, diagnostics, gRPC bootstrap, migrations, Docker builds, deployment
layout, and test wiring. Those copies gradually drift between projects.

Macro provides an opinionated starting point for that infrastructure while
leaving domain behavior, use cases, and external integrations in application
code.

> A workload should be independently runnable and testable.

## Quick Start

Install the CLI:

```bash
go install github.com/railzwaylabs/macro/cmd/macro@latest
```

Ensure `$(go env GOPATH)/bin` is in `PATH`, then create a workspace and two
projects:

```bash
macro workspace init commerce
cd commerce

macro init billing-api --type service
macro init daily-report --type job
```

Run the generated service:

```bash
cd billing-api
make run
```

Service initialization creates a minimal protobuf service contract and Buf v2
configuration, runs `buf generate` when Buf is installed, then resolves Go
modules. If Buf is unavailable, Macro keeps the project and prints the exact
commands needed to finish generation.

The service listens for gRPC traffic on `:8000` and exposes diagnostics on
`127.0.0.1:6060`. Stop it with `Ctrl+C`.

The generated job is finite and exits after its work completes:

```bash
cd ../daily-report
go run ./cmd/job
```

## What Macro Provides

| Capability | Behavior |
| --- | --- |
| Service lifecycle | Starts registered servers and stops them in reverse order |
| Worker lifecycle | Runs application-owned workers with signal-driven context cancellation |
| gRPC runtime | Provides a standard `google.golang.org/grpc` server on `:8000` |
| Structured logging | Uses Zap with a runtime-adjustable log level |
| Diagnostics | Serves pprof and log-level management on a loopback listener |
| Data-store setup | Configures GORM connections for PostgreSQL, MySQL, and SQLite |
| Project scaffolding | Generates runnable projects with Make, Docker, Git, and golangci-lint defaults |
| Module scaffolding | Creates compile-safe application/transport boundaries and an empty migration pair |
| Workspace management | Tracks related projects using normalized relative paths |
| Container builds | Generates a Dockerfile; Kubernetes and Nomad manifests are not scaffolded |

## Workload Types

| Workload | Lifecycle | Typical use |
| --- | --- | --- |
| Service | Start → handle requests → graceful shutdown | gRPC or other request/response APIs |
| Worker | Start → wait/process repeatedly → graceful shutdown | Queue consumption, streams, or database polling |
| Job | Start → execute finite work → exit | Migrations, reconciliation, or batch processing |

### Workers

Macro owns only the worker lifecycle boundary:

```go
type Worker interface {
	Run(context.Context) error
}
```

`macro.RunWorker` creates a signal-aware context, logs lifecycle events, and
calls the application worker directly. A generated worker starts with an
infrastructure-neutral implementation:

```go
type Worker struct{}

func (worker *Worker) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
```

Workers may receive work from Kafka, NATS, RabbitMQ, Redis Streams, SQS,
Pub/Sub, database polling, or another source. Consumer, message,
acknowledgement, offset, and retry semantics belong to application or
infrastructure code—not Macro.

### Jobs

A job executes finite work and exits:

```text
start → execute → exit
```

A job does not own its schedule. Schedule it with the execution environment,
such as cron or a systemd timer, Kubernetes CronJob, Nomad periodic batch, a
CI scheduler, or a cloud scheduler. Macro does not embed a scheduler in the
job runtime.

## CLI

Create a project. `service` is the default type:

```bash
macro init billing
macro init rating-worker --type worker
macro init daily-report --type job
```

Use `--module` when the Go module path differs from the project name:

```bash
macro init billing --module github.com/example/billing
```

Inside a generated project, add an application module:

```bash
cd billing
macro add module invoice
```

This creates intentionally minimal package boundaries:

```text
internal/invoice/
├── application/service.go
└── transport/grpc/handler.go

migrations/
├── <timestamp>_invoice.up.sql
└── <timestamp>_invoice.down.sql
```

The Go files provide only empty `Service`/`Handler` types and constructors.
The migrations contain comments only. Macro does not infer domain models,
repository methods, schemas, or RPC behavior from a module name.

Create and inspect a workspace, or add an existing Macro project:

```bash
macro workspace init commerce
cd commerce
macro workspace list
macro workspace add ../catalog
```

Projects created below a workspace are registered automatically. The two
manifest types have separate responsibilities:

- `macro.yaml` is the source of truth for one project's name, workload type,
  runtime, modules, and deployment settings.
- `macro.workspace.yaml` stores workspace membership as project paths. It does
  not duplicate project metadata.

```yaml
name: commerce
projects:
  - path: ./billing-api
  - path: ./daily-report
```

Use the built-in help for flags and validation rules:

```bash
macro --help
macro init --help
macro add module --help
macro workspace --help
```

Preview mutating commands without creating files, running external tools, or
changing workspace membership:

```bash
macro init billing --type service --dry-run
macro add module invoice --dry-run
```

Check the local toolchain and current project/workspace context:

```bash
macro doctor
```

Workspace listings support stable JSON output for scripts:

```bash
macro workspace list --format json
```

Generate shell completion without modifying shell configuration:

```bash
macro completion zsh > _macro
```

Run `macro --help` for canonical command discovery.

## Application Boundaries

Macro belongs in `main.go`, at the executable or composition boundary:

```text
transport
    ↓
application / use cases
    ↓
domain

infrastructure ── implements application-owned interfaces
main.go        ── wires concrete dependencies and Macro lifecycle
```

Application code should own:

- domain rules and use cases;
- repository interfaces;
- database and messaging adapters;
- the meaning and evolution of protobuf contracts and generated code;
- transport handlers.

Macro does not need to appear in the domain layer. This layout works with DDD,
hexagonal architecture, or a simpler layered design; Macro does not require a
specific application architecture.

## Service Runtime

`NewService` provides logging, gRPC, diagnostics, signal handling, and graceful
shutdown with useful defaults:

```go
package main

import (
	"fmt"
	"os"

	"github.com/railzwaylabs/macro"
)

func main() {
	service := macro.NewService("billing")
	if err := service.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

| Default | Value |
| --- | --- |
| Logger | Zap at `info` level |
| gRPC address | `:8000` |
| Diagnostics address | `127.0.0.1:6060` |
| Shutdown timeout | 10 seconds |
| Signals | `SIGINT` and `SIGTERM` |

## gRPC

Applications own protobuf contract behavior and generated Go code. For service
projects, the CLI supplies an empty initial service contract plus Buf
configuration; Macro owns the gRPC server lifecycle.

```bash
make proto
```

Workers and jobs do not receive protobuf or Buf files by default.

Register generated handlers before calling `Run`:

```go
service := macro.NewService("billing")
billingv1.RegisterBillingServiceServer(service.GRPC(), billingHandler)
return service.Run()
```

The default address is `:8000`. Override it when necessary:

```go
service := macro.NewService("billing", macro.GRPCAddress(":9000"))
```

`Service.GRPC()` returns the native `*grpc.Server`, so applications can
register reflection, health services, and other standard gRPC services. It
returns `nil` only when `macro.WithoutGRPC()` is configured. Macro does not
introduce a proprietary RPC abstraction.

## Configuration

| Option | Purpose |
| --- | --- |
| `macro.GRPCAddress(address)` | Override the default `:8000` listener |
| `macro.DebugAddress(address)` | Override the default diagnostics listener |
| `macro.ShutdownTimeout(duration)` | Set the service graceful-shutdown deadline |
| `macro.WithLogger(logger)` | Replace the default Zap logger |
| `macro.WithServer(server)` | Add another lifecycle-managed server |
| `macro.WithoutGRPC()` | Disable the default gRPC server |
| `macro.WithoutDebug()` | Disable the default diagnostics server |

## Diagnostics

The diagnostics server is enabled by default on `127.0.0.1:6060` and exposes:

```text
GET /debug/pprof/
GET /log/mode
PUT /log/mode
```

Read or change the active log level without restarting the service:

```bash
curl http://127.0.0.1:6060/log/mode

curl -X PUT \
  -H 'Content-Type: application/json' \
  -d '{"mode":"debug"}' \
  http://127.0.0.1:6060/log/mode
```

The listener defaults to loopback and is not publicly exposed. Protect the
endpoint appropriately if a different address is configured.

## Installation

Use Macro as a library:

```bash
go get github.com/railzwaylabs/macro@latest
```

Install the CLI:

```bash
go install github.com/railzwaylabs/macro/cmd/macro@latest
macro version
```

Macro currently requires Go 1.25.7 or later. Prebuilt CLI archives for Linux,
macOS, and Windows are available from
[GitHub Releases](https://github.com/railzwaylabs/macro/releases).

## Generated Projects

The CLI-generated service, worker, and job projects are independently runnable
and testable. Every project includes a README and Makefile with run, test,
build, lint, and Docker targets; services additionally include `proto`.

## Development

Run tests and static analysis:

```bash
go test ./...
golangci-lint run ./...
```

Build the CLI locally:

```bash
go build ./cmd/macro
```

GoReleaser publishes native CLI archives when a `v*` tag is pushed. The first
release line starts at `v0.1.0`.
