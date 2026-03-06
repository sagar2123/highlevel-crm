package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type TenantDB struct {
	db *gorm.DB
}

func NewTenantDB(db *gorm.DB) *TenantDB {
	return &TenantDB{db: db}
}

func (t *TenantDB) Conn(ctx context.Context) *gorm.DB {
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return t.db.WithContext(ctx)
	}
	tx := t.db.WithContext(ctx).Begin()
	tx.Exec(fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", tenantID))
	return tx
}

func (t *TenantDB) Raw() *gorm.DB {
	return t.db
}
