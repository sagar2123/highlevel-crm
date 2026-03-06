package entity

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type CustomObjectRecord struct {
	BaseEntity
	SchemaID   uuid.UUID           `gorm:"type:uuid;not null;index" json:"schema_id"`
	Schema     *CustomObjectSchema `gorm:"foreignKey:SchemaID" json:"schema,omitempty"`
	Properties datatypes.JSON      `gorm:"type:jsonb;not null;default:'{}'" json:"properties"`
}

func (CustomObjectRecord) TableName() string { return "custom_object_records" }
