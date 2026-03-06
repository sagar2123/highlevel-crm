package entity

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Contact struct {
	BaseEntity
	FirstName        string         `gorm:"type:varchar(255)" json:"first_name"`
	LastName         string         `gorm:"type:varchar(255)" json:"last_name"`
	Email            *string        `gorm:"type:varchar(320);index" json:"email,omitempty"`
	Phone            *string        `gorm:"type:varchar(50)" json:"phone,omitempty"`
	CompanyID        *uuid.UUID     `gorm:"type:uuid;index" json:"company_id,omitempty"`
	Company          *Company       `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
	Source           *string        `gorm:"type:varchar(100)" json:"source,omitempty"`
	Tags             pq.StringArray `gorm:"type:text[]" json:"tags,omitempty"`
	CustomProperties datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"custom_properties"`
}

func (Contact) TableName() string { return "contacts" }
