package database

import (
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
	"gorm.io/gorm"
)

func ActiveOnly(db *gorm.DB) *gorm.DB {
	return db.Where("lifecycle_state = ?", valueobject.LifecycleActive)
}

func NotDeleted(db *gorm.DB) *gorm.DB {
	return db.Where("lifecycle_state != ?", valueobject.LifecycleDeleted)
}

func Paginate(offset, limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset).Limit(limit)
	}
}
