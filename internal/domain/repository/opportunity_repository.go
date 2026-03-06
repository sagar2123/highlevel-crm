package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type OpportunityRepository interface {
	Create(ctx context.Context, opp *entity.Opportunity) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Opportunity, error)
	Update(ctx context.Context, opp *entity.Opportunity) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.Opportunity, int64, error)
}
