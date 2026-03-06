package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type AssociationDefinitionRepository interface {
	Create(ctx context.Context, def *entity.AssociationDefinition) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.AssociationDefinition, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.AssociationDefinition, int64, error)
}

type AssociationRepository interface {
	Create(ctx context.Context, assoc *entity.Association) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByRecord(ctx context.Context, recordID uuid.UUID, offset, limit int) ([]entity.Association, int64, error)
}
