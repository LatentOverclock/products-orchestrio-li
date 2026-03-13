package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"products-orchestio-li/backend/internal/model"
)

type MemoryStore struct {
	mu          sync.RWMutex
	nextID      int64
	nextUserID  int64
	products    map[int64]model.Product
	users       map[int64]model.User
	userByEmail map[string]int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID:      1,
		nextUserID:  1,
		products:    map[int64]model.Product{},
		users:       map[int64]model.User{},
		userByEmail: map[string]int64{},
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *MemoryStore) ListUsers(context.Context) ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		items = append(items, u)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s *MemoryStore) GetUser(_ context.Context, id int64) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	copy := u
	return &copy, nil
}

func (s *MemoryStore) CreateUser(_ context.Context, input model.CreateUserInput) (*model.User, error) {
	email := normalizeEmail(input.Email)
	if email == "" || strings.TrimSpace(input.Password) == "" {
		return nil, ErrInvalidCredentials
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.userByEmail[email]; exists {
		return nil, ErrEmailAlreadyExists
	}

	hash, err := hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := model.User{
		ID:           s.nextUserID,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.users[u.ID] = u
	s.userByEmail[email] = u.ID
	s.nextUserID++
	copy := u
	return &copy, nil
}

func (s *MemoryStore) UpdateUser(_ context.Context, id int64, input model.UpdateUserInput) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}

	if input.Email != nil {
		nextEmail := normalizeEmail(*input.Email)
		if nextEmail == "" {
			return nil, ErrInvalidCredentials
		}
		if existingID, exists := s.userByEmail[nextEmail]; exists && existingID != id {
			return nil, ErrEmailAlreadyExists
		}
		delete(s.userByEmail, u.Email)
		u.Email = nextEmail
		s.userByEmail[nextEmail] = id
	}

	if input.Password != nil {
		password := strings.TrimSpace(*input.Password)
		if password != "" {
			hash, err := hashPassword(password)
			if err != nil {
				return nil, err
			}
			u.PasswordHash = hash
		}
	}

	u.UpdatedAt = time.Now().UTC()
	s.users[id] = u
	copy := u
	return &copy, nil
}

func (s *MemoryStore) DeleteUser(_ context.Context, id int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[id]
	if !ok {
		return false, nil
	}
	delete(s.userByEmail, u.Email)
	delete(s.users, id)
	return true, nil
}

func (s *MemoryStore) AuthenticateUser(_ context.Context, email, password string) (*model.User, error) {
	normEmail := normalizeEmail(email)

	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.userByEmail[normEmail]
	if !ok {
		return nil, ErrInvalidCredentials
	}
	u := s.users[id]
	if !verifyPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	copy := u
	return &copy, nil
}

func (s *MemoryStore) EnsureUser(ctx context.Context, input model.CreateUserInput) (*model.User, error) {
	normEmail := normalizeEmail(input.Email)
	s.mu.RLock()
	id, ok := s.userByEmail[normEmail]
	if ok {
		u := s.users[id]
		s.mu.RUnlock()
		copy := u
		return &copy, nil
	}
	s.mu.RUnlock()
	return s.CreateUser(ctx, input)
}

func (s *MemoryStore) Close() {}
