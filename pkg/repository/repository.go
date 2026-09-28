package repository

import (
	"context"

	"github.com/railzwaylabs/macro/pkg/option"
)

type Repository[E any, Q any] interface {
	FindByID(context.Context, string) (E, error)
	Find(context.Context, Q, ...option.QueryOption) ([]E, error)
	Create(context.Context, E) (*E, error)
	Update(context.Context, string, E) (*E, error)
	Delete(context.Context, string) error
}
