package crm

import (
	"time"

	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type CreateRecordRequest struct {
	Properties map[string]interface{} `json:"properties" binding:"required"`
}

type UpdateRecordRequest struct {
	Properties map[string]interface{} `json:"properties" binding:"required"`
}

type RecordResponse struct {
	ID             string                 `json:"id"`
	ObjectType     string                 `json:"object_type"`
	Properties     map[string]interface{} `json:"properties"`
	LifecycleState string                 `json:"lifecycle_state"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type ListResponse struct {
	Results  []RecordResponse `json:"results"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	HasMore  bool             `json:"has_more"`
}

type CreateAssociationRequest struct {
	DefinitionID   string `json:"definition_id" binding:"required"`
	TargetRecordID string `json:"target_record_id" binding:"required"`
	TargetObjectType string `json:"target_object_type" binding:"required"`
}

type AssociationResponse struct {
	ID               string `json:"id"`
	DefinitionID     string `json:"definition_id"`
	SourceRecordID   string `json:"source_record_id"`
	TargetRecordID   string `json:"target_record_id"`
	SourceObjectType string `json:"source_object_type"`
	TargetObjectType string `json:"target_object_type"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateSchemaRequest struct {
	SingularName string                    `json:"singular_name" binding:"required"`
	PluralName   string                    `json:"plural_name" binding:"required"`
	Slug         string                    `json:"slug" binding:"required"`
	PrimaryField string                    `json:"primary_field" binding:"required"`
	Fields       []FieldDefinitionRequest  `json:"fields" binding:"required"`
}

type FieldDefinitionRequest struct {
	Key       string   `json:"key" binding:"required"`
	Label     string   `json:"label" binding:"required"`
	FieldType string   `json:"field_type" binding:"required"`
	Required  bool     `json:"required"`
	Unique    bool     `json:"unique"`
	Options   []string `json:"options,omitempty"`
}

type SchemaResponse struct {
	ID           string                   `json:"id"`
	Slug         string                   `json:"slug"`
	SingularName string                   `json:"singular_name"`
	PluralName   string                   `json:"plural_name"`
	PrimaryField string                   `json:"primary_field"`
	Fields       []FieldDefinitionRequest `json:"fields"`
	LifecycleState string                 `json:"lifecycle_state"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
}

type CreateAssociationDefinitionRequest struct {
	SourceObjectType string `json:"source_object_type" binding:"required"`
	TargetObjectType string `json:"target_object_type" binding:"required"`
	SourceLabel      string `json:"source_label" binding:"required"`
	TargetLabel      string `json:"target_label" binding:"required"`
	Cardinality      string `json:"cardinality" binding:"required"`
}

type AssociationDefinitionResponse struct {
	ID               string                  `json:"id"`
	SourceObjectType string                  `json:"source_object_type"`
	TargetObjectType string                  `json:"target_object_type"`
	SourceLabel      string                  `json:"source_label"`
	TargetLabel      string                  `json:"target_label"`
	Cardinality      valueobject.Cardinality `json:"cardinality"`
	CreatedAt        time.Time               `json:"created_at"`
}

type PipelineRequest struct {
	Name   string         `json:"name" binding:"required"`
	Stages []StageRequest `json:"stages,omitempty"`
}

type StageRequest struct {
	Name     string `json:"name" binding:"required"`
	Position int    `json:"position"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []FieldError  `json:"details,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
