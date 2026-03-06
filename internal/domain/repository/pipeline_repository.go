package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type PipelineRepository interface {
	Create(ctx context.Context, pipeline *entity.Pipeline) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Pipeline, error)
	Update(ctx context.Context, pipeline *entity.Pipeline) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.Pipeline, int64, error)
	AddStage(ctx context.Context, stage *entity.PipelineStage) error
}
