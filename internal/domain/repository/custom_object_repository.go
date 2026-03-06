package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type CustomObjectSchemaRepository interface {
	Create(ctx context.Context, schema *entity.CustomObjectSchema) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.CustomObjectSchema, error)
	GetBySlug(ctx context.Context, slug string) (*entity.CustomObjectSchema, error)
	Update(ctx context.Context, schema *entity.CustomObjectSchema) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.CustomObjectSchema, int64, error)
}

type CustomObjectRecordRepository interface {
	Create(ctx context.Context, record *entity.CustomObjectRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.CustomObjectRecord, error)
	Update(ctx context.Context, record *entity.CustomObjectRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, schemaID uuid.UUID, offset, limit int) ([]entity.CustomObjectRecord, int64, error)
}
