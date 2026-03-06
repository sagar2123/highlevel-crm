package entity

import "gorm.io/datatypes"

type Company struct {
	BaseEntity
	Name             string         `gorm:"type:varchar(500);not null" json:"name"`
	Domain           *string        `gorm:"type:varchar(255);index" json:"domain,omitempty"`
	Industry         *string        `gorm:"type:varchar(100)" json:"industry,omitempty"`
	EmployeeCount    *int           `gorm:"type:int" json:"employee_count,omitempty"`
	AnnualRevenue    *int64         `gorm:"type:bigint" json:"annual_revenue,omitempty"`
	Address          datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"address"`
	CustomProperties datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"custom_properties"`
}

func (Company) TableName() string { return "companies" }
