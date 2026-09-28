# Basic Macro Service

This example starts a Macro service with:

- a Zap logger managed by Macro;
- a gRPC server listening on Macro's default address `:8000`;
- a debug management server listening on `127.0.0.1:6060`;
- an application service isolated from Macro, gRPC, and the repository implementation;
- graceful shutdown on `SIGINT` or `SIGTERM`.

Run it from the Macro module root:

```bash
go run ./example/basic
```

Stop it with `Ctrl+C`.

Call the sample invoice RPC with `grpcurl`:

```bash
grpcurl \
  -plaintext \
  -import-path example/basic/api/proto \
  -proto billing/v1/billing.proto \
  -d '{"invoice_id":"inv-001"}' \
  localhost:8000 \
  billing.v1.BillingService/GetInvoice
```

## Architecture

There are two different concepts named "service" in this example:

| Component | Responsibility |
| --- | --- |
| `application.BillingService` | Runs billing use cases and business rules |
| generated `BillingServiceServer` | Defines the protobuf RPC contract |
| `grpcapi.BillingHandler` | Adapts an RPC request to the application service |
| `macro.Service` | Owns process, server, and graceful-shutdown lifecycle |

The request flow is:

```text
gRPC caller
    │
    ▼
BillingHandler                 transport adapter
    │
    ▼
application.BillingService    business use case
    │
    ▼
InvoiceRepository interface   application-owned port
    │
    ▼
memory.InvoiceRepository      infrastructure adapter
```

`BillingService` does not import Macro, protobuf, gRPC, or Gorm. It only
depends on its `InvoiceRepository` interface. Consequently, a database adapter
can replace the in-memory repository without changing the use case.

`main.go` is the **composition root**. It is the only place that knows how all
layers are assembled:

```go
invoiceRepository := memory.NewInvoiceRepository(/* seed data */)
billingService := application.NewBillingService(invoiceRepository)
billingHandler := grpcapi.NewBillingHandler(billingService)

rpcServer := macrogrpc.New("", func(server *grpc.Server) {
	billingv1.RegisterBillingServiceServer(server, billingHandler)
})

app := macro.New(
	macro.Name("billing"),
	macro.WithLogger(log),
	macro.WithServer(rpcServer),
	macro.WithServer(debugServer),
)

return app.Run()
```

Macro's composition-root pattern is inspired by Go Micro: the toolkit is used
at the executable boundary while business use cases remain
framework-independent.

The management server exposes:

```text
GET /debug/pprof/
GET /log/mode
PUT /log/mode
```

Use a custom debug address when needed:

```go
debugServer := debug.New(debug.Config{
	Address: "127.0.0.1:6061",
	Service: "billing",
}, log)
```

The application entrypoint only configures the service and calls `Run`:

```go
app := macro.New(
	macro.Name("billing"),
	macro.WithLogger(log),
	macro.WithServer(rpcServer),
)

if err := app.Run(); err != nil {
	return err
}
```

Macro handles `SIGINT`/`SIGTERM`, server startup, the default 10-second
graceful-shutdown timeout, and reverse-order server shutdown. Override the
timeout with `macro.ShutdownTimeout(...)` when needed.

## Generate protobuf code

The example contract is located at
`api/proto/billing/v1/billing.proto`. Install the Go protobuf plugins once:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

With `protoc` installed, run this command from the Macro module root:

```bash
protoc -I . \
  --go_out=. \
  --go_opt=module=github.com/railzwaylabs/macro \
  --go-grpc_out=. \
  --go-grpc_opt=module=github.com/railzwaylabs/macro \
  example/basic/api/proto/billing/v1/billing.proto
```

Generated files will be written to:

```text
example/basic/gen/billing/v1/
├── billing.pb.go
└── billing_grpc.pb.go
```

## Register the generated service

The callback passed to `macrogrpc.New` is where generated protobuf services
are registered:

```go
rpcServer := macrogrpc.New("", func(server *grpc.Server) {
	billingv1.RegisterBillingServiceServer(server, billingHandler)
})
```

Pass an explicit address to override the default:

```go
rpcServer := macrogrpc.New(":9000", registerServices)
```

The application owns its protobuf contract and generated handler interfaces;
Macro owns the server lifecycle. See these files for the complete example:

```text
internal/application/billing.go           business use case and repository port
internal/repository/memory/invoice.go      repository adapter
internal/transport/grpc/billing.go         protobuf/gRPC adapter
main.go                                    dependency wiring and Macro runtime
```
