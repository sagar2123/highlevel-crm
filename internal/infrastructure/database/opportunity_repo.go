package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type opportunityRepo struct {
	tenant *TenantDB
}

func NewOpportunityRepository(tenant *TenantDB) *opportunityRepo {
	return &opportunityRepo{tenant: tenant}
}

func (r *opportunityRepo) Create(ctx context.Context, opp *entity.Opportunity) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(opp).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *opportunityRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Opportunity, error) {
	tx := r.tenant.Conn(ctx)
	var opp entity.Opportunity
	err := tx.Scopes(NotDeleted).Where("id = ?", id).First(&opp).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &opp, nil
}

func (r *opportunityRepo) Update(ctx context.Context, opp *entity.Opportunity) error {
	tx := r.tenant.Conn(ctx)
	opp.UpdatedAt = time.Now()
	if err := tx.Save(opp).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *opportunityRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	now := time.Now()
	err := tx.Model(&entity.Opportunity{}).Where("id = ?", id).Updates(map[string]interface{}{
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

func (r *opportunityRepo) List(ctx context.Context, offset, limit int) ([]entity.Opportunity, int64, error) {
	tx := r.tenant.Conn(ctx)
	var opps []entity.Opportunity
	var total int64

	tx.Model(&entity.Opportunity{}).Scopes(ActiveOnly).Count(&total)
	err := tx.Scopes(ActiveOnly, Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&opps).Error
	tx.Commit()
	return opps, total, err
}
