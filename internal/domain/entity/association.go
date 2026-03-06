package entity

import (
	"time"

	"github.com/google/uuid"
)

type Association struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	DefinitionID   uuid.UUID `gorm:"type:uuid;not null;index" json:"definition_id"`
	SourceRecordID uuid.UUID `gorm:"type:uuid;not null;index" json:"source_record_id"`
	TargetRecordID uuid.UUID `gorm:"type:uuid;not null;index" json:"target_record_id"`
	CreatedAt      time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (Association) TableName() string { return "associations" }
