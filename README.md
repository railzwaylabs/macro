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
> `v0.4.0` is under development, and APIs may evolve before the first stable
> release.

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

The service listens for gRPC traffic on `:9000` and exposes diagnostics on
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
| gRPC runtime | Provides a standard `google.golang.org/grpc` server on `:9000` |
| gRPC-Gateway | Opt-in HTTP/JSON routes generated from `google.api.http` annotations |
| Dependency wiring | Uses generated module registration with Fx at the composition boundary |
| HTTP runtime | Optionally serves an application-owned `net/http` handler on `:8080` |
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
├── module.go
├── application/service.go
└── transport/grpc/handler.go

migrations/
├── <timestamp>_invoice.up.sql
└── <timestamp>_invoice.down.sql
```

The Go files provide only empty `Service`/`Handler` types, constructors, and a
predictable `Module` registration entry point. Macro updates the generated
`internal/modules/modules.go` registry, so `cmd/service/main.go` does not need
manual edits when another module is added.
The migrations contain comments only. Macro does not infer domain models,
repository methods, schemas, or RPC behavior from a module name.

Add a standard protobuf/gRPC API to an existing module:

```bash
macro add grpc invoice
```

Add the same native gRPC API plus annotated HTTP/JSON routes:

```bash
macro add grpc invoice --gateway
```

Both commands update the generated module registry, run Buf, and resolve Go
modules. The gateway flag also enables the HTTP listener in `macro.yaml`; no
manual edit to `cmd/service/main.go` is required.

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
macro add grpc --help
macro workspace --help
```

Preview mutating commands without creating files, running external tools, or
changing workspace membership:

```bash
macro init billing --type service --dry-run
macro add module invoice --dry-run
macro add grpc invoice --gateway --dry-run
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
| HTTP | Disabled; `:8080` when enabled |
| gRPC address | `:9000` |
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

### Native gRPC and HTTP/JSON

`macro add grpc product --gateway` generates this transport flow from one
protobuf contract:

```text
HTTP/JSON                 gRPC client
    │                         │
    ▼                         ▼
grpc-gateway ─────────► gRPC handler
                            │
                            ▼
                       application
                            │
                            ▼
                          domain
```

The generated layout keeps contracts and generated artifacts separate:

```text
api/proto/catalogue/v1/product.proto
gen/catalogue/v1/product.pb.go
gen/catalogue/v1/product_grpc.pb.go
gen/catalogue/v1/product.pb.gw.go
```

Routes remain owned by `google.api.http` annotations in the proto file. Macro
uses grpc-gateway's default status mapping, including `InvalidArgument` → 400,
`NotFound` → 404, and `Unavailable` → 503.

The generated gateway currently uses the official in-process registration
adapter. Both transports call the same application service, but gateway calls
bypass network-level gRPC interceptors. Native gRPC requests still use Macro's
logging and recovery interceptors. This keeps local startup deterministic and
avoids a second set of HTTP business handlers.

Generated services expose the standard gRPC health protocol automatically.
Reflection is opt-in because it exposes service metadata and may be unsuitable
for a public production listener:

```go
app := macro.NewService("catalogue", macro.WithGRPCReflection())
```

Test the generated project normally; transport tests can use `bufconn` for
gRPC and `httptest` with the gateway mux without binding external ports:

```bash
go test ./...
go run ./cmd/service
```

Then call `localhost:9000` with a generated gRPC client, or use HTTP:

```bash
curl -X POST http://localhost:8080/v1/products \
  -H 'Content-Type: application/json' \
  -d '{"name":"Keyboard"}'
```

PostgreSQL/Testcontainers scaffolding and OpenTelemetry exporters remain
separate concerns until persistence and telemetry generation are explicitly
enabled. The manifest keeps the conventional OTLP gRPC endpoint
`localhost:4317`; it is not an application listener.

The default address is `:9000`. Override it when necessary:

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
| `macro.GRPCAddress(address)` | Override the default `:9000` listener |
| `macro.WithHTTP(handler)` | Enable HTTP with an application-owned `net/http` handler |
| `macro.WithGateway(options...)` | Enable HTTP using an official grpc-gateway `ServeMux` |
| `macro.HTTPAddress(address)` | Override the default HTTP `:8080` listener |
| `macro.WithModules(modules...)` | Register generated dependency-wiring modules |
| `macro.WithGRPCReflection()` | Enable standard gRPC server reflection |
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

GoReleaser publishes native CLI archives when a `v*` tag is pushed.
