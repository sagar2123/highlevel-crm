package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type companyRepo struct {
	tenant *TenantDB
}

func NewCompanyRepository(tenant *TenantDB) *companyRepo {
	return &companyRepo{tenant: tenant}
}

func (r *companyRepo) Create(ctx context.Context, company *entity.Company) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(company).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *companyRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Company, error) {
	tx := r.tenant.Conn(ctx)
	var company entity.Company
	err := tx.Scopes(NotDeleted).Where("id = ?", id).First(&company).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *companyRepo) Update(ctx context.Context, company *entity.Company) error {
	tx := r.tenant.Conn(ctx)
	company.UpdatedAt = time.Now()
	if err := tx.Save(company).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *companyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	now := time.Now()
	err := tx.Model(&entity.Company{}).Where("id = ?", id).Updates(map[string]interface{}{
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

func (r *companyRepo) List(ctx context.Context, offset, limit int) ([]entity.Company, int64, error) {
	tx := r.tenant.Conn(ctx)
	var companies []entity.Company
	var total int64

	tx.Model(&entity.Company{}).Scopes(ActiveOnly).Count(&total)
	err := tx.Scopes(ActiveOnly, Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&companies).Error
	tx.Commit()
	return companies, total, err
}
