package entity

import (
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
	"gorm.io/datatypes"
)

type CustomObjectSchema struct {
	BaseEntity
	Slug         string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_cos_location_slug" json:"slug"`
	SingularName string         `gorm:"type:varchar(255);not null" json:"singular_name"`
	PluralName   string         `gorm:"type:varchar(255);not null" json:"plural_name"`
	PrimaryField string         `gorm:"type:varchar(100);not null" json:"primary_field"`
	Fields       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"fields"`
}

type FieldDefinition struct {
	Key       string              `json:"key"`
	Label     string              `json:"label"`
	FieldType valueobject.FieldType `json:"field_type"`
	Required  bool                `json:"required"`
	Unique    bool                `json:"unique"`
	Options   []string            `json:"options,omitempty"`
}

func (CustomObjectSchema) TableName() string { return "custom_object_schemas" }
