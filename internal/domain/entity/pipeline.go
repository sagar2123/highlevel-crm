package entity

import (
	"time"

	"github.com/google/uuid"
)

type Pipeline struct {
	BaseEntity
	Name   string          `gorm:"type:varchar(255);not null" json:"name"`
	Stages []PipelineStage `gorm:"foreignKey:PipelineID;constraint:OnDelete:CASCADE" json:"stages,omitempty"`
}

type PipelineStage struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PipelineID uuid.UUID `gorm:"type:uuid;not null;index" json:"pipeline_id"`
	TenantID   uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name       string    `gorm:"type:varchar(255);not null" json:"name"`
	Position   int       `gorm:"type:int;not null;default:0" json:"position"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (Pipeline) TableName() string      { return "pipelines" }
func (PipelineStage) TableName() string { return "pipeline_stages" }
