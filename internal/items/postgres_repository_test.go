package items

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresRepositoryCreateUsesParameterizedQuery(t *testing.T) {
	t.Parallel()

	db := &recordingPostgresDB{}
	repository := &PostgresRepository{db: db}
	item := Item{
		ID:        "018fb3b2-0f9d-4f59-8a63-7ef4bb812345",
		Name:      "Example item'); DROP TABLE items; --",
		CreatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
	}

	if err := repository.Create(context.Background(), item); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if !strings.Contains(db.execQuery, "VALUES ($1, $2, $3, $4)") {
		t.Fatalf("expected parameterized insert query, got %q", db.execQuery)
	}

	if strings.Contains(db.execQuery, item.Name) {
		t.Fatalf("expected query not to contain untrusted item name, got %q", db.execQuery)
	}

	if len(db.execArgs) != 4 {
		t.Fatalf("expected 4 query args, got %d", len(db.execArgs))
	}

	if db.execArgs[1] != item.Name {
		t.Fatalf("expected item name to be passed as argument, got %#v", db.execArgs[1])
	}
}

func TestPostgresRepositoryFindByID(t *testing.T) {
	t.Parallel()

	expected := Item{
		ID:        "018fb3b2-0f9d-4f59-8a63-7ef4bb812345",
		Name:      "Example item",
		CreatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
	}
	db := &recordingPostgresDB{
		row: rowFunc(func(dest ...any) error {
			*(dest[0].(*string)) = expected.ID
			*(dest[1].(*string)) = expected.Name
			*(dest[2].(*time.Time)) = expected.CreatedAt
			*(dest[3].(*time.Time)) = expected.UpdatedAt
			return nil
		}),
	}
	repository := &PostgresRepository{db: db}

	item, err := repository.FindByID(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("expected find to succeed, got %v", err)
	}

	if item != expected {
		t.Fatalf("expected item %+v, got %+v", expected, item)
	}

	if !strings.Contains(db.queryRowQuery, "WHERE id = $1") {
		t.Fatalf("expected parameterized select query, got %q", db.queryRowQuery)
	}
}

func TestPostgresRepositoryFindByIDMapsNoRowsToNotFound(t *testing.T) {
	t.Parallel()

	repository := &PostgresRepository{
		db: &recordingPostgresDB{
			row: rowFunc(func(...any) error {
				return pgx.ErrNoRows
			}),
		},
	}

	_, err := repository.FindByID(context.Background(), "018fb3b2-0f9d-4f59-8a63-7ef4bb812345")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestIsSafeIntegrationSchemaName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "safe", value: "test_items_123456789", want: true},
		{name: "wrong prefix", value: "items_123456789", want: false},
		{name: "contains hyphen", value: "test_items_123-456", want: false},
		{name: "contains semicolon", value: "test_items_123;DROP_SCHEMA", want: false},
		{name: "contains uppercase", value: "test_items_ABC", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := isSafeIntegrationSchemaName(tt.value); got != tt.want {
				t.Fatalf("expected %t, got %t", tt.want, got)
			}
		})
	}
}

type recordingPostgresDB struct {
	execQuery     string
	execArgs      []any
	queryRowQuery string
	queryRowArgs  []any
	row           pgx.Row
}

func (db *recordingPostgresDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	db.execQuery = query
	db.execArgs = args
	return pgconn.CommandTag{}, nil
}

func (db *recordingPostgresDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.queryRowQuery = query
	db.queryRowArgs = args
	return db.row
}

type rowFunc func(...any) error

func (f rowFunc) Scan(dest ...any) error {
	return f(dest...)
}
