package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type customObjectSchemaRepo struct {
	tenant *TenantDB
}

func NewCustomObjectSchemaRepository(tenant *TenantDB) *customObjectSchemaRepo {
	return &customObjectSchemaRepo{tenant: tenant}
}

func (r *customObjectSchemaRepo) Create(ctx context.Context, schema *entity.CustomObjectSchema) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(schema).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *customObjectSchemaRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.CustomObjectSchema, error) {
	tx := r.tenant.Conn(ctx)
	var schema entity.CustomObjectSchema
	err := tx.Scopes(NotDeleted).Where("id = ?", id).First(&schema).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func (r *customObjectSchemaRepo) GetBySlug(ctx context.Context, slug string) (*entity.CustomObjectSchema, error) {
	tx := r.tenant.Conn(ctx)
	var schema entity.CustomObjectSchema
	err := tx.Scopes(NotDeleted).Where("slug = ?", slug).First(&schema).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func (r *customObjectSchemaRepo) Update(ctx context.Context, schema *entity.CustomObjectSchema) error {
	tx := r.tenant.Conn(ctx)
	schema.UpdatedAt = time.Now()
	if err := tx.Save(schema).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *customObjectSchemaRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	now := time.Now()
	err := tx.Model(&entity.CustomObjectSchema{}).Where("id = ?", id).Updates(map[string]interface{}{
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

func (r *customObjectSchemaRepo) List(ctx context.Context, offset, limit int) ([]entity.CustomObjectSchema, int64, error) {
	tx := r.tenant.Conn(ctx)
	var schemas []entity.CustomObjectSchema
	var total int64

	tx.Model(&entity.CustomObjectSchema{}).Scopes(ActiveOnly).Count(&total)
	err := tx.Scopes(ActiveOnly, Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&schemas).Error
	tx.Commit()
	return schemas, total, err
}

type customObjectRecordRepo struct {
	tenant *TenantDB
}

func NewCustomObjectRecordRepository(tenant *TenantDB) *customObjectRecordRepo {
	return &customObjectRecordRepo{tenant: tenant}
}

func (r *customObjectRecordRepo) Create(ctx context.Context, record *entity.CustomObjectRecord) error {
	tx := r.tenant.Conn(ctx)
	if err := tx.Create(record).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *customObjectRecordRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.CustomObjectRecord, error) {
	tx := r.tenant.Conn(ctx)
	var record entity.CustomObjectRecord
	err := tx.Scopes(NotDeleted).Where("id = ?", id).First(&record).Error
	tx.Commit()
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *customObjectRecordRepo) Update(ctx context.Context, record *entity.CustomObjectRecord) error {
	tx := r.tenant.Conn(ctx)
	record.UpdatedAt = time.Now()
	if err := tx.Save(record).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (r *customObjectRecordRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx := r.tenant.Conn(ctx)
	now := time.Now()
	err := tx.Model(&entity.CustomObjectRecord{}).Where("id = ?", id).Updates(map[string]interface{}{
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

func (r *customObjectRecordRepo) List(ctx context.Context, schemaID uuid.UUID, offset, limit int) ([]entity.CustomObjectRecord, int64, error) {
	tx := r.tenant.Conn(ctx)
	var records []entity.CustomObjectRecord
	var total int64

	tx.Model(&entity.CustomObjectRecord{}).
		Where("schema_id = ?", schemaID).
		Scopes(ActiveOnly).Count(&total)
	err := tx.Where("schema_id = ?", schemaID).
		Scopes(ActiveOnly, Paginate(offset, limit)).
		Order("created_at DESC").
		Find(&records).Error
	tx.Commit()
	return records, total, err
}
