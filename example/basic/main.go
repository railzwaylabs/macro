package main

import (
	"fmt"
	"os"

	"github.com/railzwaylabs/macro"
	"github.com/railzwaylabs/macro/debug"
	billingv1 "github.com/railzwaylabs/macro/example/basic/gen/billing/v1"
	"github.com/railzwaylabs/macro/example/basic/internal/application"
	"github.com/railzwaylabs/macro/example/basic/internal/repository/memory"
	grpcapi "github.com/railzwaylabs/macro/example/basic/internal/transport/grpc"
	macrologger "github.com/railzwaylabs/macro/logger"
	macrogrpc "github.com/railzwaylabs/macro/server/transport/grpc"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	log, err := macrologger.New(macrologger.Config{
		Service:     "billing",
		Mode:        "info",
		Development: true,
	})
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	// Infrastructure adapter. Replace this with a Postgres/MySQL repository
	// without changing the application service.
	invoiceRepository := memory.NewInvoiceRepository(application.Invoice{
		ID:         "inv-001",
		CustomerID: "customer-001",
		Amount:     150_000,
		Currency:   "IDR",
		Status:     "issued",
	})

	// Application service: this owns the business use case and has no
	// dependency on Macro, gRPC, protobuf, or a concrete database.
	billingService := application.NewBillingService(invoiceRepository)

	// Transport adapter: converts protobuf requests/responses to application
	// input/output.
	billingHandler := grpcapi.NewBillingHandler(billingService)

	// Macro is used only at the composition root to own process and server
	// lifecycle. An empty address uses macrogrpc.DefaultAddress (:8000).
	rpcServer := macrogrpc.New("", func(server *grpc.Server) {
		billingv1.RegisterBillingServiceServer(server, billingHandler)
	})
	debugServer := debug.New(debug.Config{
		Service: "billing",
	}, log)

	app := macro.New(
		macro.Name("billing"),
		macro.WithLogger(log),
		macro.WithServer(rpcServer),
		macro.WithServer(debugServer),
	)

	return app.Run()
}
