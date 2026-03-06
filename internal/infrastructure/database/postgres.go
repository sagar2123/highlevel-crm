package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sagar2123/highlevel-crm/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresConnection(cfg config.DB) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdle)
	sqlDB.SetMaxOpenConns(cfg.MaxOpen)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	log.Println("connected to postgres")
	return db, nil
}

func SetTenantContext(db *gorm.DB, ctx context.Context) *gorm.DB {
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return db
	}
	return db.Session(&gorm.Session{NewDB: true}).
		Exec("SET LOCAL app.current_tenant_id = ?", tenantID)
}

func WithTenant(db *gorm.DB, ctx context.Context) *gorm.DB {
	tenantID, ok := ctx.Value("tenant_id").(string)
	if !ok || tenantID == "" {
		return db.WithContext(ctx)
	}
	tx := db.WithContext(ctx).Begin()
	tx.Exec("SET LOCAL app.current_tenant_id = ?", tenantID)
	return tx
}
