package store

import (
	"context"
	"time"

	"products-orchestio-li/backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	s := &PostgresStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
	CREATE TABLE IF NOT EXISTS products (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		purchase_link TEXT,
		shop_link TEXT,
		booqable_link TEXT,
		manual_link TEXT,
		inspection_link TEXT,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		CONSTRAINT products_status_check CHECK (status IN ('mafo','write-manual','all-done'))
	);
	CREATE INDEX IF NOT EXISTS products_status_idx ON products(status);
	`)
	return err
}

func (s *PostgresStore) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.pool.Ping(ctx)
}

func scanProduct(row pgx.Row) (*model.Product, error) {
	var p model.Product
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.PurchaseLink,
		&p.ShopLink,
		&p.BooqableLink,
		&p.ManualLink,
		&p.InspectionLink,
		&p.Description,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *PostgresStore) ListProducts(ctx context.Context) ([]model.Product, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, purchase_link, shop_link, booqable_link, manual_link, inspection_link, description, status, created_at, updated_at
		FROM products
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []model.Product{}
	for rows.Next() {
		var p model.Product
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.PurchaseLink,
			&p.ShopLink,
			&p.BooqableLink,
			&p.ManualLink,
			&p.InspectionLink,
			&p.Description,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *PostgresStore) GetProduct(ctx context.Context, id int64) (*model.Product, error) {
	return scanProduct(s.pool.QueryRow(ctx, `
		SELECT id, name, purchase_link, shop_link, booqable_link, manual_link, inspection_link, description, status, created_at, updated_at
		FROM products
		WHERE id = $1
	`, id))
}

func (s *PostgresStore) CreateProduct(ctx context.Context, input model.ProductInput) (*model.Product, error) {
	return scanProduct(s.pool.QueryRow(ctx, `
		INSERT INTO products (
			name, purchase_link, shop_link, booqable_link, manual_link, inspection_link, description, status
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, name, purchase_link, shop_link, booqable_link, manual_link, inspection_link, description, status, created_at, updated_at
	`,
		input.Name,
		input.PurchaseLink,
		input.ShopLink,
		input.BooqableLink,
		input.ManualLink,
		input.InspectionLink,
		input.Description,
		input.Status,
	))
}

func (s *PostgresStore) UpdateProduct(ctx context.Context, id int64, input model.ProductInput) (*model.Product, error) {
	return scanProduct(s.pool.QueryRow(ctx, `
		UPDATE products
		SET name=$2,
			purchase_link=$3,
			shop_link=$4,
			booqable_link=$5,
			manual_link=$6,
			inspection_link=$7,
			description=$8,
			status=$9,
			updated_at=NOW()
		WHERE id=$1
		RETURNING id, name, purchase_link, shop_link, booqable_link, manual_link, inspection_link, description, status, created_at, updated_at
	`,
		id,
		input.Name,
		input.PurchaseLink,
		input.ShopLink,
		input.BooqableLink,
		input.ManualLink,
		input.InspectionLink,
		input.Description,
		input.Status,
	))
}

func (s *PostgresStore) DeleteProduct(ctx context.Context, id int64) (bool, error) {
	result, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return result.RowsAffected() > 0, nil
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}
