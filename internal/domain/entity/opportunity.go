package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Opportunity struct {
	BaseEntity
	Name              string         `gorm:"type:varchar(500);not null" json:"name"`
	PipelineID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"pipeline_id"`
	Pipeline          *Pipeline      `gorm:"foreignKey:PipelineID" json:"pipeline,omitempty"`
	StageID           uuid.UUID      `gorm:"type:uuid;not null;index" json:"stage_id"`
	Stage             *PipelineStage `gorm:"foreignKey:StageID" json:"stage,omitempty"`
	ContactID         *uuid.UUID     `gorm:"type:uuid;index" json:"contact_id,omitempty"`
	Contact           *Contact       `gorm:"foreignKey:ContactID" json:"contact,omitempty"`
	CompanyID         *uuid.UUID     `gorm:"type:uuid;index" json:"company_id,omitempty"`
	Company           *Company       `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
	MonetaryValue     *int64         `gorm:"type:bigint" json:"monetary_value,omitempty"`
	Currency          string         `gorm:"type:varchar(3);default:'USD'" json:"currency"`
	ExpectedCloseDate *time.Time     `json:"expected_close_date,omitempty"`
	AssignedTo        *uuid.UUID     `gorm:"type:uuid;index" json:"assigned_to,omitempty"`
	CustomProperties  datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"custom_properties"`
}

func (Opportunity) TableName() string { return "opportunities" }
