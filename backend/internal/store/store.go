package store

import (
	"context"
	"errors"

	"products-orchestio-li/backend/internal/model"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailAlreadyExists = errors.New("email already exists")

type Store interface {
	Health(context.Context) error

	ListProducts(context.Context) ([]model.Product, error)
	GetProduct(context.Context, int64) (*model.Product, error)
	CreateProduct(context.Context, model.ProductInput) (*model.Product, error)
	UpdateProduct(context.Context, int64, model.ProductInput) (*model.Product, error)
	DeleteProduct(context.Context, int64) (bool, error)

	ListUsers(context.Context) ([]model.User, error)
	GetUser(context.Context, int64) (*model.User, error)
	CreateUser(context.Context, model.CreateUserInput) (*model.User, error)
	UpdateUser(context.Context, int64, model.UpdateUserInput) (*model.User, error)
	DeleteUser(context.Context, int64) (bool, error)
	AuthenticateUser(context.Context, string, string) (*model.User, error)
	EnsureUser(context.Context, model.CreateUserInput) (*model.User, error)

	Close()
}
