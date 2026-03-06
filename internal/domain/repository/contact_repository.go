package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type ContactRepository interface {
	Create(ctx context.Context, contact *entity.Contact) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Contact, error)
	Update(ctx context.Context, contact *entity.Contact) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]entity.Contact, int64, error)
}
