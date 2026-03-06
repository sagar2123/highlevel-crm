package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type CompanyRepository interface {
	Create(ctx context.Context, company *entity.Company) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Company, error)
	Update(ctx context.Context, company *entity.Company) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.Company, int64, error)
}
