package items

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestServiceCreateItem(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 23, 10, 30, 0, 0, time.FixedZone("UTC-3", -3*60*60))
	repository := &serviceFakeRepository{}
	service := NewService(repository)
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "018fb3b2-0f9d-4f59-8a63-7ef4bb812345", nil }

	item, err := service.Create(context.Background(), "  Example item  ")
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if item.ID != "018fb3b2-0f9d-4f59-8a63-7ef4bb812345" {
		t.Fatalf("expected generated id, got %q", item.ID)
	}

	if item.Name != "Example item" {
		t.Fatalf("expected trimmed name, got %q", item.Name)
	}

	if item.CreatedAt.Location() != time.UTC {
		t.Fatalf("expected created_at in UTC, got %s", item.CreatedAt.Location())
	}

	if !item.CreatedAt.Equal(item.UpdatedAt) {
		t.Fatalf("expected created_at and updated_at to match, got %s and %s", item.CreatedAt, item.UpdatedAt)
	}

	if repository.created.Name != "Example item" {
		t.Fatalf("expected repository to receive trimmed name, got %q", repository.created.Name)
	}
}

func TestServiceCreateRejectsInvalidNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "empty",
			input:     "",
			wantError: "name is required",
		},
		{
			name:      "blank",
			input:     "   ",
			wantError: "name is required",
		},
		{
			name:      "too long",
			input:     strings.Repeat("a", 201),
			wantError: "name must be at most 200 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(&serviceFakeRepository{})
			_, err := service.Create(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			var validationErr ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected ValidationError, got %T", err)
			}

			if validationErr.Message != tt.wantError {
				t.Fatalf("expected %q, got %q", tt.wantError, validationErr.Message)
			}
		})
	}
}

func TestServiceGetItem(t *testing.T) {
	t.Parallel()

	expected := Item{
		ID:        "018fb3b2-0f9d-4f59-8a63-7ef4bb812345",
		Name:      "Example item",
		CreatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
	}
	service := NewService(&serviceFakeRepository{found: expected})

	item, err := service.Get(context.Background(), "  018fb3b2-0f9d-4f59-8a63-7ef4bb812345  ")
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}

	if item != expected {
		t.Fatalf("expected item %+v, got %+v", expected, item)
	}
}

func TestServiceGetRejectsInvalidID(t *testing.T) {
	t.Parallel()

	service := NewService(&serviceFakeRepository{})

	_, err := service.Get(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}

	if validationErr.Field != "id" {
		t.Fatalf("expected id field, got %q", validationErr.Field)
	}
}

func TestServiceGetReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := NewService(&serviceFakeRepository{findErr: ErrNotFound})

	_, err := service.Get(context.Background(), "018fb3b2-0f9d-4f59-8a63-7ef4bb812345")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

type serviceFakeRepository struct {
	created Item
	found   Item
	findErr error
}

func (r *serviceFakeRepository) Create(_ context.Context, item Item) error {
	r.created = item
	return nil
}

func (r *serviceFakeRepository) FindByID(_ context.Context, _ string) (Item, error) {
	if r.findErr != nil {
		return Item{}, r.findErr
	}

	return r.found, nil
}
