package main

import (
	"fmt"
	"os"

	"github.com/railzwaylabs/macro"
	billingv1 "github.com/railzwaylabs/macro/example/basic/gen/billing/v1"
	"github.com/railzwaylabs/macro/example/basic/internal/application"
	"github.com/railzwaylabs/macro/example/basic/internal/infrastructure/repository/memory"
	grpcapi "github.com/railzwaylabs/macro/example/basic/internal/transport/grpc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
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

	// Macro provides the logger, gRPC server, diagnostics server, signals, and
	// graceful shutdown. The application only registers its transport handler.
	app := macro.NewService("billing")
	billingv1.RegisterBillingServiceServer(app.GRPC(), billingHandler)

	return app.Run()
}
