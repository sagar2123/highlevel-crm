package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type BaseEntity struct {
	ID             uuid.UUID                  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID                  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	LifecycleState valueobject.LifecycleState `gorm:"type:lifecycle_state;not null;default:'active'" json:"lifecycle_state"`
	CreatedBy      *uuid.UUID                 `gorm:"type:uuid" json:"created_by,omitempty"`
	UpdatedBy      *uuid.UUID                 `gorm:"type:uuid" json:"updated_by,omitempty"`
	CreatedAt      time.Time                  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time                  `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt      *time.Time                 `gorm:"index" json:"deleted_at,omitempty"`
}
