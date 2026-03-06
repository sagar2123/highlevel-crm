package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type AssociationDefinition struct {
	ID               uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID              `gorm:"type:uuid;not null;index" json:"tenant_id"`
	SourceObjectType string                 `gorm:"type:varchar(100);not null" json:"source_object_type"`
	TargetObjectType string                 `gorm:"type:varchar(100);not null" json:"target_object_type"`
	SourceLabel      string                 `gorm:"type:varchar(255);not null" json:"source_label"`
	TargetLabel      string                 `gorm:"type:varchar(255);not null" json:"target_label"`
	Cardinality      valueobject.Cardinality `gorm:"type:cardinality_type;not null" json:"cardinality"`
	CreatedAt        time.Time              `gorm:"not null;default:now()" json:"created_at"`
}

func (AssociationDefinition) TableName() string { return "association_definitions" }
