package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"products-orchestio-li/backend/internal/model"
)

type MemoryStore struct {
	mu       sync.RWMutex
	nextID   int64
	products map[int64]model.Product
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID:   1,
		products: map[int64]model.Product{},
	}
}

func (s *MemoryStore) Health(context.Context) error { return nil }

func (s *MemoryStore) ListProducts(context.Context) ([]model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.Product, 0, len(s.products))
	for _, p := range s.products {
		items = append(items, p)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID > items[j].ID })
	return items, nil
}

func (s *MemoryStore) GetProduct(_ context.Context, id int64) (*model.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	if !ok {
		return nil, ErrNotFound
	}
	copy := p
	return &copy, nil
}

func (s *MemoryStore) CreateProduct(_ context.Context, input model.ProductInput) (*model.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	p := model.Product{
		ID:             s.nextID,
		Name:           input.Name,
		PurchaseLink:   input.PurchaseLink,
		ShopLink:       input.ShopLink,
		BooqableLink:   input.BooqableLink,
		ManualLink:     input.ManualLink,
		InspectionLink: input.InspectionLink,
		Description:    input.Description,
		Status:         input.Status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.products[p.ID] = p
	s.nextID++
	copy := p
	return &copy, nil
}

func (s *MemoryStore) UpdateProduct(_ context.Context, id int64, input model.ProductInput) (*model.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return nil, ErrNotFound
	}
	p.Name = input.Name
	p.PurchaseLink = input.PurchaseLink
	p.ShopLink = input.ShopLink
	p.BooqableLink = input.BooqableLink
	p.ManualLink = input.ManualLink
	p.InspectionLink = input.InspectionLink
	p.Description = input.Description
	p.Status = input.Status
	p.UpdatedAt = time.Now().UTC()
	s.products[id] = p
	copy := p
	return &copy, nil
}

func (s *MemoryStore) DeleteProduct(_ context.Context, id int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return false, nil
	}
	delete(s.products, id)
	return true, nil
}

func (s *MemoryStore) Close() {}
