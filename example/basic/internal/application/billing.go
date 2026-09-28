package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrInvoiceNotFound = errors.New("invoice not found")

// Invoice is an application model. It deliberately does not depend on
// protobuf, gRPC, Gorm, or Macro.
type Invoice struct {
	ID         string
	CustomerID string
	Amount     int64
	Currency   string
	Status     string
}

// InvoiceRepository is a port owned by the application layer.
type InvoiceRepository interface {
	FindByID(context.Context, string) (*Invoice, error)
}

// BillingService implements billing use cases.
type BillingService struct {
	invoices InvoiceRepository
}

func NewBillingService(invoices InvoiceRepository) *BillingService {
	return &BillingService{invoices: invoices}
}

func (s *BillingService) GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error) {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return nil, fmt.Errorf("invoice ID is required")
	}

	invoice, err := s.invoices.FindByID(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("find invoice %q: %w", invoiceID, err)
	}
	if invoice == nil {
		return nil, ErrInvoiceNotFound
	}

	return invoice, nil
}
