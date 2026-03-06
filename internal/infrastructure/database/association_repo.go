package database

import (
	"context"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
)

type associationDefinitionRepo struct {
	tenant *TenantDB
}

func NewAssociationDefinitionRepository(tenant *TenantDB) *associationDefinitionRepo {
	return &associationDefinitionRepo{tenant: tenant}
}

func (r *associationDefinitionRepo) Create(ctx context.Context, def *entity.AssociationDefinition) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(def).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *associationDefinitionRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.AssociationDefinition, error) {
	tx := r.tenant.Conn(ctx)
	var def entity.AssociationDefinition
	err := tx.Where("id = ?", id).First(&def).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *associationDefinitionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Where("id = ?", id).Delete(&entity.AssociationDefinition{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *associationDefinitionRepo) List(ctx context.Context, offset, limit int) ([]entity.AssociationDefinition, int64, error) {
	tx := r.tenant.Conn(ctx)
	var defs []entity.AssociationDefinition
	var total int64

	tx.Model(&entity.AssociationDefinition{}).Count(&total)
	err := tx.Scopes(Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&defs).Error
	tx.Commit()
	return defs, total, err
}

type associationRepo struct {
	tenant *TenantDB
}

func NewAssociationRepository(tenant *TenantDB) *associationRepo {
	return &associationRepo{tenant: tenant}
}

func (r *associationRepo) Create(ctx context.Context, assoc *entity.Association) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(assoc).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *associationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Where("id = ?", id).Delete(&entity.Association{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *associationRepo) ListByRecord(ctx context.Context, recordID uuid.UUID, offset, limit int) ([]entity.Association, int64, error) {
	tx := r.tenant.Conn(ctx)
	var assocs []entity.Association
	var total int64

	tx.Model(&entity.Association{}).
		Where("source_record_id = ? OR target_record_id = ?", recordID, recordID).
		Count(&total)
	err := tx.Where("source_record_id = ? OR target_record_id = ?", recordID, recordID).
		Scopes(Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&assocs).Error
	tx.Commit()
	return assocs, total, err
}
