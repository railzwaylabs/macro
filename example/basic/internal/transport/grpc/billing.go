package grpc

import (
	"context"
	"errors"

	billingv1 "github.com/railzwaylabs/macro/example/basic/gen/billing/v1"
	"github.com/railzwaylabs/macro/example/basic/internal/application"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BillingHandler adapts the protobuf transport to the application service.
type BillingHandler struct {
	billingv1.UnimplementedBillingServiceServer
	service *application.BillingService
}

func NewBillingHandler(service *application.BillingService) *BillingHandler {
	return &BillingHandler{service: service}
}

func (h *BillingHandler) GetInvoice(
	ctx context.Context,
	req *billingv1.GetInvoiceRequest,
) (*billingv1.GetInvoiceResponse, error) {
	invoice, err := h.service.GetInvoice(ctx, req.GetInvoiceId())
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvoiceNotFound):
			return nil, status.Error(codes.NotFound, "invoice not found")
		default:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	return &billingv1.GetInvoiceResponse{
		Invoice: &billingv1.Invoice{
			Id:          invoice.ID,
			CustomerId:  invoice.CustomerID,
			AmountMinor: invoice.Amount,
			Currency:    invoice.Currency,
			Status:      toProtoStatus(invoice.Status),
		},
	}, nil
}

func toProtoStatus(value string) billingv1.InvoiceStatus {
	switch value {
	case "draft":
		return billingv1.InvoiceStatus_INVOICE_STATUS_DRAFT
	case "issued":
		return billingv1.InvoiceStatus_INVOICE_STATUS_ISSUED
	case "paid":
		return billingv1.InvoiceStatus_INVOICE_STATUS_PAID
	case "void":
		return billingv1.InvoiceStatus_INVOICE_STATUS_VOID
	default:
		return billingv1.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED
	}
}
