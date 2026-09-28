# Macro

Macro is a Go service toolkit inspired by
[Go Micro](https://github.com/micro/go-micro). It provides a consistent way
for Railzway services and other Go applications to assemble infrastructure,
start servers, and manage their process lifecycle.

Macro is intended for teams that want:

- a small and uniform service bootstrap;
- common defaults for gRPC, logging, debugging, and data stores;
- graceful startup and shutdown managed in one place;
- application and business logic that remains independent of the framework;
- reusable conventions across multiple Go services.

Applications use Macro at their executable entrypoint:

```go
app := macro.New(
	macro.Name("billing"),
	macro.WithLogger(log),
	macro.WithServer(rpcServer),
)

return app.Run()
```

Repositories, application services, and transport handlers remain owned by
the application. Macro only wires shared capabilities and manages their
lifecycle.

See [`example/basic`](./example/basic) for a complete gRPC service example.
