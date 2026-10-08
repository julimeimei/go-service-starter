package items

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var repositoryTracer = otel.Tracer("github.com/julimeimei/go-service-starter/internal/items")

type postgresDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresRepository struct {
	db postgresDB
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, item Item) error {
	if ctx == nil {
		return fmt.Errorf("items repository context is required")
	}

	if r == nil || r.db == nil {
		return fmt.Errorf("items repository database is required")
	}

	ctx, span := repositoryTracer.Start(
		ctx,
		"items.postgres.create",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", "INSERT"),
			attribute.String("db.collection.name", "items"),
		),
	)
	defer span.End()

	const query = `
INSERT INTO items (id, name, created_at, updated_at)
VALUES ($1, $2, $3, $4)`

	if _, err := r.db.Exec(ctx, query, item.ID, item.Name, item.CreatedAt, item.UpdatedAt); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "insert item failed")
		return fmt.Errorf("insert item: %w", err)
	}

	return nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Item, error) {
	if ctx == nil {
		return Item{}, fmt.Errorf("items repository context is required")
	}

	if r == nil || r.db == nil {
		return Item{}, fmt.Errorf("items repository database is required")
	}

	ctx, span := repositoryTracer.Start(
		ctx,
		"items.postgres.find_by_id",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system.name", "postgresql"),
			attribute.String("db.operation.name", "SELECT"),
			attribute.String("db.collection.name", "items"),
		),
	)
	defer span.End()

	const query = `
SELECT id, name, created_at, updated_at
FROM items
WHERE id = $1`

	var item Item
	if err := r.db.QueryRow(ctx, query, id).Scan(&item.ID, &item.Name, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Item{}, ErrNotFound
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, "select item by id failed")
		return Item{}, fmt.Errorf("select item by id: %w", err)
	}

	return item, nil
}
