package items

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegrationPostgresRepositoryCreateAndFindByID(t *testing.T) {
	databaseURL, ok := os.LookupEnv("INTEGRATION_DATABASE_URL")
	if !ok || strings.TrimSpace(databaseURL) == "" {
		t.Skip("set INTEGRATION_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool := openIsolatedIntegrationPool(t, ctx, databaseURL)
	defer pool.Close()

	repository := NewPostgresRepository(pool)
	now := time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC)
	expected := Item{
		ID:        "018fb3b2-0f9d-4f59-8a63-7ef4bb812345",
		Name:      "Integration item",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repository.Create(ctx, expected); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	item, err := repository.FindByID(ctx, expected.ID)
	if err != nil {
		t.Fatalf("expected find to succeed, got %v", err)
	}

	if item.ID != expected.ID {
		t.Fatalf("expected id %q, got %q", expected.ID, item.ID)
	}

	if item.Name != expected.Name {
		t.Fatalf("expected name %q, got %q", expected.Name, item.Name)
	}

	if !item.CreatedAt.Equal(expected.CreatedAt) {
		t.Fatalf("expected created_at %s, got %s", expected.CreatedAt, item.CreatedAt)
	}

	if !item.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("expected updated_at %s, got %s", expected.UpdatedAt, item.UpdatedAt)
	}
}

func TestIntegrationPostgresRepositoryFindByIDReturnsNotFound(t *testing.T) {
	databaseURL, ok := os.LookupEnv("INTEGRATION_DATABASE_URL")
	if !ok || strings.TrimSpace(databaseURL) == "" {
		t.Skip("set INTEGRATION_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool := openIsolatedIntegrationPool(t, ctx, databaseURL)
	defer pool.Close()

	repository := NewPostgresRepository(pool)

	_, err := repository.FindByID(ctx, "018fb3b2-0f9d-4f59-8a63-7ef4bb812345")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func openIsolatedIntegrationPool(t *testing.T, ctx context.Context, databaseURL string) *pgxpool.Pool {
	t.Helper()

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal("failed to parse integration database URL")
	}

	schemaName := fmt.Sprintf("test_items_%d", time.Now().UnixNano())
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal("failed to open integration database")
	}
	t.Cleanup(func() {
		defer adminPool.Close()

		if !isSafeIntegrationSchemaName(schemaName) {
			return
		}

		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		_, _ = adminPool.Exec(cleanupCtx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
	})

	createIntegrationSchema(t, ctx, adminPool, schemaName)

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal("failed to open isolated integration database pool")
	}

	return pool
}

func createIntegrationSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName string) {
	t.Helper()

	if !isSafeIntegrationSchemaName(schemaName) {
		t.Fatalf("unsafe integration schema name %q", schemaName)
	}

	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schemaName)); err != nil {
		t.Fatal("failed to create integration schema")
	}

	createItemsTable := fmt.Sprintf(`
CREATE TABLE %s.items (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`, schemaName)

	if _, err := pool.Exec(ctx, createItemsTable); err != nil {
		t.Fatal("failed to create integration items table")
	}
}

func isSafeIntegrationSchemaName(name string) bool {
	if !strings.HasPrefix(name, "test_items_") {
		return false
	}

	for _, char := range name {
		if char >= 'a' && char <= 'z' {
			continue
		}

		if char >= '0' && char <= '9' {
			continue
		}

		if char == '_' {
			continue
		}

		return false
	}

	return true
}
