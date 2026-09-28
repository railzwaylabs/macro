package memory

import (
	"context"
	"sync"

	"github.com/railzwaylabs/macro/example/basic/internal/application"
)

// InvoiceRepository is an in-memory adapter used only to keep the example
// runnable without a database.
type InvoiceRepository struct {
	mu       sync.RWMutex
	invoices map[string]*application.Invoice
}

func NewInvoiceRepository(invoices ...application.Invoice) *InvoiceRepository {
	repository := &InvoiceRepository{invoices: make(map[string]*application.Invoice, len(invoices))}
	for i := range invoices {
		invoice := invoices[i]
		repository.invoices[invoice.ID] = &invoice
	}
	return repository
}

func (r *InvoiceRepository) FindByID(_ context.Context, id string) (*application.Invoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	invoice, ok := r.invoices[id]
	if !ok {
		return nil, nil
	}

	result := *invoice
	return &result, nil
}
