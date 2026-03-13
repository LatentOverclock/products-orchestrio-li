package store

import (
	"context"
	"errors"

	"products-orchestio-li/backend/internal/model"
)

var ErrNotFound = errors.New("product not found")

type Store interface {
	Health(context.Context) error
	ListProducts(context.Context) ([]model.Product, error)
	GetProduct(context.Context, int64) (*model.Product, error)
	CreateProduct(context.Context, model.ProductInput) (*model.Product, error)
	UpdateProduct(context.Context, int64, model.ProductInput) (*model.Product, error)
	DeleteProduct(context.Context, int64) (bool, error)
	Close()
}
