package items

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Repository interface {
	Create(context.Context, Item) error
	FindByID(context.Context, string) (Item, error)
}

type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		now:        time.Now,
		newID:      NewUUID,
	}
}

func (s *Service) Create(ctx context.Context, name string) (Item, error) {
	if ctx == nil {
		return Item{}, fmt.Errorf("items context is required")
	}

	if s == nil || s.repository == nil {
		return Item{}, fmt.Errorf("items repository is required")
	}

	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return Item{}, err
	}

	id, err := s.newID()
	if err != nil {
		return Item{}, fmt.Errorf("generate item id: %w", err)
	}

	now := s.now().UTC()
	item := Item{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repository.Create(ctx, item); err != nil {
		return Item{}, fmt.Errorf("create item: %w", err)
	}

	return item, nil
}

func (s *Service) Get(ctx context.Context, id string) (Item, error) {
	if ctx == nil {
		return Item{}, fmt.Errorf("items context is required")
	}

	if s == nil || s.repository == nil {
		return Item{}, fmt.Errorf("items repository is required")
	}

	id = strings.TrimSpace(id)
	if err := validateID(id); err != nil {
		return Item{}, err
	}

	item, err := s.repository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Item{}, ErrNotFound
		}

		return Item{}, fmt.Errorf("get item: %w", err)
	}

	return item, nil
}
