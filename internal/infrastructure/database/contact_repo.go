package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type contactRepo struct {
	tenant *TenantDB
}

func NewContactRepository(tenant *TenantDB) *contactRepo {
	return &contactRepo{tenant: tenant}
}

func (r *contactRepo) Create(ctx context.Context, contact *entity.Contact) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(contact).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *contactRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Contact, error) {
	tx := r.tenant.Conn(ctx)
	var contact entity.Contact
	err := tx.Scopes(NotDeleted).Where("id = ?", id).First(&contact).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *contactRepo) Update(ctx context.Context, contact *entity.Contact) error {
	tx := r.tenant.Conn(ctx)
	contact.UpdatedAt = time.Now()
	if err := tx.Save(contact).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *contactRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	now := time.Now()
	err := tx.Model(&entity.Contact{}).Where("id = ?", id).Updates(map[string]interface{}{
		"lifecycle_state": valueobject.LifecycleDeleted,
		"deleted_at":      now,
		"updated_at":      now,
	}).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *contactRepo) List(ctx context.Context, offset, limit int) ([]entity.Contact, int64, error) {
	tx := r.tenant.Conn(ctx)
	var contacts []entity.Contact
	var total int64

	tx.Model(&entity.Contact{}).Scopes(ActiveOnly).Count(&total)
	err := tx.Scopes(ActiveOnly, Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&contacts).Error
	tx.Commit()
	return contacts, total, err
}
