package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
	"gorm.io/gorm"
)

type pipelineRepo struct {
	tenant *TenantDB
}

func NewPipelineRepository(tenant *TenantDB) *pipelineRepo {
	return &pipelineRepo{tenant: tenant}
}

func (r *pipelineRepo) Create(ctx context.Context, pipeline *entity.Pipeline) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(pipeline).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *pipelineRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Pipeline, error) {
	tx := r.tenant.Conn(ctx)
	var pipeline entity.Pipeline
	err := tx.Scopes(NotDeleted).
		Preload("Stages", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Where("id = ?", id).First(&pipeline).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

func (r *pipelineRepo) Update(ctx context.Context, pipeline *entity.Pipeline) error {
	tx := r.tenant.Conn(ctx)
	pipeline.UpdatedAt = time.Now()
	if err := tx.Save(pipeline).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *pipelineRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	now := time.Now()
	err := tx.Model(&entity.Pipeline{}).Where("id = ?", id).Updates(map[string]interface{}{
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

func (r *pipelineRepo) List(ctx context.Context, offset, limit int) ([]entity.Pipeline, int64, error) {
	tx := r.tenant.Conn(ctx)
	var pipelines []entity.Pipeline
	var total int64

	tx.Model(&entity.Pipeline{}).Scopes(ActiveOnly).Count(&total)
	err := tx.Scopes(ActiveOnly, Paginate(offset, limit)).
		Preload("Stages", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Order("created_at DESC").
		Find(&pipelines).Error
	tx.Commit()
	return pipelines, total, err
}

func (r *pipelineRepo) AddStage(ctx context.Context, stage *entity.PipelineStage) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(stage).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
