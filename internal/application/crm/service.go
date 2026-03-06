package crm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/repository"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
	"github.com/sagar2123/highlevel-crm/internal/infrastructure/elasticsearch"
	"gorm.io/datatypes"
)

type Service struct {
	contacts      repository.ContactRepository
	companies     repository.CompanyRepository
	opportunities repository.OpportunityRepository
	pipelines     repository.PipelineRepository
	schemas       repository.CustomObjectSchemaRepository
	records       repository.CustomObjectRecordRepository
	assocDefs     repository.AssociationDefinitionRepository
	assocs        repository.AssociationRepository
	search        repository.SearchRepository
	sync          *elasticsearch.SyncService
}

func NewService(
	contacts repository.ContactRepository,
	companies repository.CompanyRepository,
	opportunities repository.OpportunityRepository,
	pipelines repository.PipelineRepository,
	schemas repository.CustomObjectSchemaRepository,
	records repository.CustomObjectRecordRepository,
	assocDefs repository.AssociationDefinitionRepository,
	assocs repository.AssociationRepository,
	search repository.SearchRepository,
	sync *elasticsearch.SyncService,
) *Service {
	return &Service{
		contacts:      contacts,
		companies:     companies,
		opportunities: opportunities,
		pipelines:     pipelines,
		schemas:       schemas,
		records:       records,
		assocDefs:     assocDefs,
		assocs:        assocs,
		search:        search,
		sync:          sync,
	}
}

func (s *Service) CreateRecord(ctx context.Context, objectType string, req CreateRecordRequest) (*RecordResponse, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)

	switch objectType {
	case "contacts":
		contact := contactFromProperties(req.Properties, tenantID)
		if err := s.contacts.Create(ctx, contact); err != nil {
			return nil, err
		}
		resp := toRecordResponse(objectType, contact.ID.String(), contact.LifecycleState, contact.CreatedAt, contact.UpdatedAt, contactToProperties(contact))
		s.sync.IndexDocument(ctx, objectType, contact.ID.String(), resp.Properties)
		return resp, nil

	case "companies":
		company := companyFromProperties(req.Properties, tenantID)
		if err := s.companies.Create(ctx, company); err != nil {
			return nil, err
		}
		resp := toRecordResponse(objectType, company.ID.String(), company.LifecycleState, company.CreatedAt, company.UpdatedAt, companyToProperties(company))
		s.sync.IndexDocument(ctx, objectType, company.ID.String(), resp.Properties)
		return resp, nil

	case "opportunities":
		opp := opportunityFromProperties(req.Properties, tenantID)
		if err := s.opportunities.Create(ctx, opp); err != nil {
			return nil, err
		}
		resp := toRecordResponse(objectType, opp.ID.String(), opp.LifecycleState, opp.CreatedAt, opp.UpdatedAt, opportunityToProperties(opp))
		s.sync.IndexDocument(ctx, objectType, opp.ID.String(), resp.Properties)
		return resp, nil

	default:
		return s.createCustomRecord(ctx, objectType, req, tenantID)
	}
}

func (s *Service) createCustomRecord(ctx context.Context, objectType string, req CreateRecordRequest, tenantID string) (*RecordResponse, error) {
	schema, err := s.schemas.GetBySlug(ctx, objectType)
	if err != nil {
		return nil, fmt.Errorf("unknown object type: %s", objectType)
	}

	tID, _ := uuid.Parse(tenantID)
	propsJSON, _ := json.Marshal(req.Properties)

	record := &entity.CustomObjectRecord{
		BaseEntity: entity.BaseEntity{
			TenantID:       tID,
			LifecycleState: valueobject.LifecycleActive,
		},
		SchemaID:   schema.ID,
		Properties: datatypes.JSON(propsJSON),
	}

	if err := s.records.Create(ctx, record); err != nil {
		return nil, err
	}

	resp := toRecordResponse(objectType, record.ID.String(), record.LifecycleState, record.CreatedAt, record.UpdatedAt, req.Properties)
	s.sync.IndexDocument(ctx, "custom_objects", record.ID.String(), map[string]interface{}{
		"id":              record.ID.String(),
		"tenant_id":       tenantID,
		"schema_id":       schema.ID.String(),
		"object_type":     objectType,
		"properties":      req.Properties,
		"lifecycle_state": string(record.LifecycleState),
		"created_at":      record.CreatedAt,
		"updated_at":      record.UpdatedAt,
	})
	return resp, nil
}

func (s *Service) GetRecord(ctx context.Context, objectType string, id uuid.UUID) (*RecordResponse, error) {
	switch objectType {
	case "contacts":
		c, err := s.contacts.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return toRecordResponse(objectType, c.ID.String(), c.LifecycleState, c.CreatedAt, c.UpdatedAt, contactToProperties(c)), nil

	case "companies":
		c, err := s.companies.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return toRecordResponse(objectType, c.ID.String(), c.LifecycleState, c.CreatedAt, c.UpdatedAt, companyToProperties(c)), nil

	case "opportunities":
		o, err := s.opportunities.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return toRecordResponse(objectType, o.ID.String(), o.LifecycleState, o.CreatedAt, o.UpdatedAt, opportunityToProperties(o)), nil

	default:
		record, err := s.records.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		var props map[string]interface{}
		json.Unmarshal(record.Properties, &props)
		return toRecordResponse(objectType, record.ID.String(), record.LifecycleState, record.CreatedAt, record.UpdatedAt, props), nil
	}
}

func (s *Service) UpdateRecord(ctx context.Context, objectType string, id uuid.UUID, req UpdateRecordRequest) (*RecordResponse, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)

	switch objectType {
	case "contacts":
		existing, err := s.contacts.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		applyContactUpdates(existing, req.Properties)
		if err := s.contacts.Update(ctx, existing); err != nil {
			return nil, err
		}
		props := contactToProperties(existing)
		s.sync.IndexDocument(ctx, objectType, id.String(), props)
		return toRecordResponse(objectType, id.String(), existing.LifecycleState, existing.CreatedAt, existing.UpdatedAt, props), nil

	case "companies":
		existing, err := s.companies.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		applyCompanyUpdates(existing, req.Properties)
		if err := s.companies.Update(ctx, existing); err != nil {
			return nil, err
		}
		props := companyToProperties(existing)
		s.sync.IndexDocument(ctx, objectType, id.String(), props)
		return toRecordResponse(objectType, id.String(), existing.LifecycleState, existing.CreatedAt, existing.UpdatedAt, props), nil

	case "opportunities":
		existing, err := s.opportunities.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		applyOpportunityUpdates(existing, req.Properties)
		if err := s.opportunities.Update(ctx, existing); err != nil {
			return nil, err
		}
		props := opportunityToProperties(existing)
		s.sync.IndexDocument(ctx, objectType, id.String(), props)
		return toRecordResponse(objectType, id.String(), existing.LifecycleState, existing.CreatedAt, existing.UpdatedAt, props), nil

	default:
		record, err := s.records.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		propsJSON, _ := json.Marshal(req.Properties)
		record.Properties = datatypes.JSON(propsJSON)
		if err := s.records.Update(ctx, record); err != nil {
			return nil, err
		}
		s.sync.IndexDocument(ctx, "custom_objects", id.String(), map[string]interface{}{
			"id":              id.String(),
			"tenant_id":       tenantID,
			"properties":      req.Properties,
			"lifecycle_state": string(record.LifecycleState),
			"updated_at":      record.UpdatedAt,
		})
		return toRecordResponse(objectType, id.String(), record.LifecycleState, record.CreatedAt, record.UpdatedAt, req.Properties), nil
	}
}

func (s *Service) DeleteRecord(ctx context.Context, objectType string, id uuid.UUID) error {
	var err error
	switch objectType {
	case "contacts":
		err = s.contacts.Delete(ctx, id)
	case "companies":
		err = s.companies.Delete(ctx, id)
	case "opportunities":
		err = s.opportunities.Delete(ctx, id)
	default:
		err = s.records.Delete(ctx, id)
		objectType = "custom_objects"
	}
	if err != nil {
		return err
	}
	s.sync.RemoveDocument(ctx, objectType, id.String())
	return nil
}

func (s *Service) ListRecords(ctx context.Context, objectType string, page, pageSize int) (*ListResponse, error) {
	offset := (page - 1) * pageSize

	switch objectType {
	case "contacts":
		items, total, err := s.contacts.List(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
		results := make([]RecordResponse, 0, len(items))
		for _, c := range items {
			results = append(results, *toRecordResponse(objectType, c.ID.String(), c.LifecycleState, c.CreatedAt, c.UpdatedAt, contactToProperties(&c)))
		}
		return &ListResponse{Results: results, Total: total, Page: page, PageSize: pageSize, HasMore: int64(page*pageSize) < total}, nil

	case "companies":
		items, total, err := s.companies.List(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
		results := make([]RecordResponse, 0, len(items))
		for _, c := range items {
			results = append(results, *toRecordResponse(objectType, c.ID.String(), c.LifecycleState, c.CreatedAt, c.UpdatedAt, companyToProperties(&c)))
		}
		return &ListResponse{Results: results, Total: total, Page: page, PageSize: pageSize, HasMore: int64(page*pageSize) < total}, nil

	case "opportunities":
		items, total, err := s.opportunities.List(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
		results := make([]RecordResponse, 0, len(items))
		for _, o := range items {
			results = append(results, *toRecordResponse(objectType, o.ID.String(), o.LifecycleState, o.CreatedAt, o.UpdatedAt, opportunityToProperties(&o)))
		}
		return &ListResponse{Results: results, Total: total, Page: page, PageSize: pageSize, HasMore: int64(page*pageSize) < total}, nil

	case "pipelines":
		items, total, err := s.pipelines.List(ctx, offset, pageSize)
		if err != nil {
			return nil, err
		}
		results := make([]RecordResponse, 0, len(items))
		for _, p := range items {
			props := map[string]interface{}{"name": p.Name, "stages": p.Stages}
			results = append(results, *toRecordResponse(objectType, p.ID.String(), p.LifecycleState, p.CreatedAt, p.UpdatedAt, props))
		}
		return &ListResponse{Results: results, Total: total, Page: page, PageSize: pageSize, HasMore: int64(page*pageSize) < total}, nil

	default:
		schema, err := s.schemas.GetBySlug(ctx, objectType)
		if err != nil {
			return nil, fmt.Errorf("unknown object type: %s", objectType)
		}
		items, total, err := s.records.List(ctx, schema.ID, offset, pageSize)
		if err != nil {
			return nil, err
		}
		results := make([]RecordResponse, 0, len(items))
		for _, r := range items {
			var props map[string]interface{}
			json.Unmarshal(r.Properties, &props)
			results = append(results, *toRecordResponse(objectType, r.ID.String(), r.LifecycleState, r.CreatedAt, r.UpdatedAt, props))
		}
		return &ListResponse{Results: results, Total: total, Page: page, PageSize: pageSize, HasMore: int64(page*pageSize) < total}, nil
	}
}

func (s *Service) SearchRecords(ctx context.Context, objectType string, req valueobject.SearchRequest) (*valueobject.SearchResult, error) {
	indexName := objectType
	if !valueobject.IsBuiltInObjectType(objectType) {
		indexName = "custom_objects"
	}
	return s.search.Search(ctx, indexName, req)
}

func (s *Service) ArchiveRecord(ctx context.Context, objectType string, id uuid.UUID) error {
	return s.updateLifecycleState(ctx, objectType, id, valueobject.LifecycleArchived)
}

func (s *Service) RestoreRecord(ctx context.Context, objectType string, id uuid.UUID) error {
	return s.updateLifecycleState(ctx, objectType, id, valueobject.LifecycleActive)
}

func (s *Service) updateLifecycleState(ctx context.Context, objectType string, id uuid.UUID, state valueobject.LifecycleState) error {
	switch objectType {
	case "contacts":
		c, err := s.contacts.GetByID(ctx, id)
		if err != nil {
			return err
		}
		c.LifecycleState = state
		if state == valueobject.LifecycleDeleted {
			now := time.Now()
			c.DeletedAt = &now
		} else {
			c.DeletedAt = nil
		}
		return s.contacts.Update(ctx, c)

	case "companies":
		c, err := s.companies.GetByID(ctx, id)
		if err != nil {
			return err
		}
		c.LifecycleState = state
		if state == valueobject.LifecycleDeleted {
			now := time.Now()
			c.DeletedAt = &now
		} else {
			c.DeletedAt = nil
		}
		return s.companies.Update(ctx, c)

	case "opportunities":
		o, err := s.opportunities.GetByID(ctx, id)
		if err != nil {
			return err
		}
		o.LifecycleState = state
		if state == valueobject.LifecycleDeleted {
			now := time.Now()
			o.DeletedAt = &now
		} else {
			o.DeletedAt = nil
		}
		return s.opportunities.Update(ctx, o)

	default:
		r, err := s.records.GetByID(ctx, id)
		if err != nil {
			return err
		}
		r.LifecycleState = state
		if state == valueobject.LifecycleDeleted {
			now := time.Now()
			r.DeletedAt = &now
		} else {
			r.DeletedAt = nil
		}
		return s.records.Update(ctx, r)
	}
}

func (s *Service) CreateSchema(ctx context.Context, req CreateSchemaRequest) (*SchemaResponse, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)
	tID, _ := uuid.Parse(tenantID)

	fields := make([]entity.FieldDefinition, 0, len(req.Fields))
	for _, f := range req.Fields {
		fields = append(fields, entity.FieldDefinition{
			Key:       f.Key,
			Label:     f.Label,
			FieldType: valueobject.FieldType(f.FieldType),
			Required:  f.Required,
			Unique:    f.Unique,
			Options:   f.Options,
		})
	}

	fieldsJSON, _ := json.Marshal(fields)
	schema := &entity.CustomObjectSchema{
		BaseEntity: entity.BaseEntity{
			TenantID:       tID,
			LifecycleState: valueobject.LifecycleActive,
		},
		Slug:         req.Slug,
		SingularName: req.SingularName,
		PluralName:   req.PluralName,
		PrimaryField: req.PrimaryField,
		Fields:       datatypes.JSON(fieldsJSON),
	}

	if err := s.schemas.Create(ctx, schema); err != nil {
		return nil, err
	}

	return &SchemaResponse{
		ID:             schema.ID.String(),
		Slug:           schema.Slug,
		SingularName:   schema.SingularName,
		PluralName:     schema.PluralName,
		PrimaryField:   schema.PrimaryField,
		Fields:         req.Fields,
		LifecycleState: string(schema.LifecycleState),
		CreatedAt:      schema.CreatedAt,
		UpdatedAt:      schema.UpdatedAt,
	}, nil
}

func (s *Service) GetSchema(ctx context.Context, id uuid.UUID) (*SchemaResponse, error) {
	schema, err := s.schemas.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return schemaToResponse(schema), nil
}

func (s *Service) ListSchemas(ctx context.Context, page, pageSize int) ([]SchemaResponse, int64, error) {
	offset := (page - 1) * pageSize
	schemas, total, err := s.schemas.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	results := make([]SchemaResponse, 0, len(schemas))
	for _, sc := range schemas {
		results = append(results, *schemaToResponse(&sc))
	}
	return results, total, nil
}

func (s *Service) UpdateSchema(ctx context.Context, id uuid.UUID, req CreateSchemaRequest) (*SchemaResponse, error) {
	schema, err := s.schemas.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	fields := make([]entity.FieldDefinition, 0, len(req.Fields))
	for _, f := range req.Fields {
		fields = append(fields, entity.FieldDefinition{
			Key:       f.Key,
			Label:     f.Label,
			FieldType: valueobject.FieldType(f.FieldType),
			Required:  f.Required,
			Unique:    f.Unique,
			Options:   f.Options,
		})
	}
	fieldsJSON, _ := json.Marshal(fields)

	schema.SingularName = req.SingularName
	schema.PluralName = req.PluralName
	schema.PrimaryField = req.PrimaryField
	schema.Fields = datatypes.JSON(fieldsJSON)

	if err := s.schemas.Update(ctx, schema); err != nil {
		return nil, err
	}
	return schemaToResponse(schema), nil
}

func (s *Service) DeleteSchema(ctx context.Context, id uuid.UUID) error {
	return s.schemas.Delete(ctx, id)
}

func (s *Service) CreateAssociationDefinition(ctx context.Context, req CreateAssociationDefinitionRequest) (*AssociationDefinitionResponse, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)
	tID, _ := uuid.Parse(tenantID)

	def := &entity.AssociationDefinition{
		TenantID:         tID,
		SourceObjectType: req.SourceObjectType,
		TargetObjectType: req.TargetObjectType,
		SourceLabel:      req.SourceLabel,
		TargetLabel:      req.TargetLabel,
		Cardinality:      valueobject.Cardinality(req.Cardinality),
	}

	if err := s.assocDefs.Create(ctx, def); err != nil {
		return nil, err
	}

	return &AssociationDefinitionResponse{
		ID:               def.ID.String(),
		SourceObjectType: def.SourceObjectType,
		TargetObjectType: def.TargetObjectType,
		SourceLabel:      def.SourceLabel,
		TargetLabel:      def.TargetLabel,
		Cardinality:      def.Cardinality,
		CreatedAt:        def.CreatedAt,
	}, nil
}

func (s *Service) ListAssociationDefinitions(ctx context.Context, page, pageSize int) ([]AssociationDefinitionResponse, int64, error) {
	offset := (page - 1) * pageSize
	defs, total, err := s.assocDefs.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	results := make([]AssociationDefinitionResponse, 0, len(defs))
	for _, d := range defs {
		results = append(results, AssociationDefinitionResponse{
			ID:               d.ID.String(),
			SourceObjectType: d.SourceObjectType,
			TargetObjectType: d.TargetObjectType,
			SourceLabel:      d.SourceLabel,
			TargetLabel:      d.TargetLabel,
			Cardinality:      d.Cardinality,
			CreatedAt:        d.CreatedAt,
		})
	}
	return results, total, nil
}

func (s *Service) DeleteAssociationDefinition(ctx context.Context, id uuid.UUID) error {
	return s.assocDefs.Delete(ctx, id)
}

func (s *Service) CreateAssociation(ctx context.Context, objectType string, recordID uuid.UUID, req CreateAssociationRequest) (*AssociationResponse, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)
	tID, _ := uuid.Parse(tenantID)
	defID, _ := uuid.Parse(req.DefinitionID)
	targetID, _ := uuid.Parse(req.TargetRecordID)

	assoc := &entity.Association{
		TenantID:       tID,
		DefinitionID:   defID,
		SourceRecordID: recordID,
		TargetRecordID: targetID,
	}

	if err := s.assocs.Create(ctx, assoc); err != nil {
		return nil, err
	}

	return &AssociationResponse{
		ID:               assoc.ID.String(),
		DefinitionID:     assoc.DefinitionID.String(),
		SourceRecordID:   assoc.SourceRecordID.String(),
		TargetRecordID:   assoc.TargetRecordID.String(),
		SourceObjectType: objectType,
		TargetObjectType: req.TargetObjectType,
		CreatedAt:        assoc.CreatedAt,
	}, nil
}

func (s *Service) ListAssociations(ctx context.Context, recordID uuid.UUID, page, pageSize int) ([]AssociationResponse, int64, error) {
	offset := (page - 1) * pageSize
	assocs, total, err := s.assocs.ListByRecord(ctx, recordID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}
	results := make([]AssociationResponse, 0, len(assocs))
	for _, a := range assocs {
		results = append(results, AssociationResponse{
			ID:             a.ID.String(),
			DefinitionID:   a.DefinitionID.String(),
			SourceRecordID: a.SourceRecordID.String(),
			TargetRecordID: a.TargetRecordID.String(),
			CreatedAt:      a.CreatedAt,
		})
	}
	return results, total, nil
}

func (s *Service) DeleteAssociation(ctx context.Context, id uuid.UUID) error {
	return s.assocs.Delete(ctx, id)
}

func (s *Service) CreatePipeline(ctx context.Context, req PipelineRequest) (*RecordResponse, error) {
	tenantID, _ := ctx.Value("tenant_id").(string)
	tID, _ := uuid.Parse(tenantID)

	pipeline := &entity.Pipeline{
		BaseEntity: entity.BaseEntity{
			TenantID:       tID,
			LifecycleState: valueobject.LifecycleActive,
		},
		Name: req.Name,
	}

	for i, s := range req.Stages {
		pipeline.Stages = append(pipeline.Stages, entity.PipelineStage{
			TenantID: tID,
			Name:       s.Name,
			Position:   i,
		})
	}

	if err := s.pipelines.Create(ctx, pipeline); err != nil {
		return nil, err
	}

	props := map[string]interface{}{"name": pipeline.Name, "stages": pipeline.Stages}
	return toRecordResponse("pipelines", pipeline.ID.String(), pipeline.LifecycleState, pipeline.CreatedAt, pipeline.UpdatedAt, props), nil
}

func (s *Service) GetPipeline(ctx context.Context, id uuid.UUID) (*RecordResponse, error) {
	p, err := s.pipelines.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	props := map[string]interface{}{"name": p.Name, "stages": p.Stages}
	return toRecordResponse("pipelines", p.ID.String(), p.LifecycleState, p.CreatedAt, p.UpdatedAt, props), nil
}

func (s *Service) AddPipelineStage(ctx context.Context, pipelineID uuid.UUID, req StageRequest) error {
	tenantID, _ := ctx.Value("tenant_id").(string)
	tID, _ := uuid.Parse(tenantID)

	stage := &entity.PipelineStage{
		PipelineID: pipelineID,
		TenantID: tID,
		Name:       req.Name,
		Position:   req.Position,
	}
	return s.pipelines.AddStage(ctx, stage)
}

func toRecordResponse(objectType, id string, state valueobject.LifecycleState, createdAt, updatedAt time.Time, props map[string]interface{}) *RecordResponse {
	return &RecordResponse{
		ID:             id,
		ObjectType:     objectType,
		Properties:     props,
		LifecycleState: string(state),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

func schemaToResponse(s *entity.CustomObjectSchema) *SchemaResponse {
	var fields []FieldDefinitionRequest
	json.Unmarshal(s.Fields, &fields)
	return &SchemaResponse{
		ID:             s.ID.String(),
		Slug:           s.Slug,
		SingularName:   s.SingularName,
		PluralName:     s.PluralName,
		PrimaryField:   s.PrimaryField,
		Fields:         fields,
		LifecycleState: string(s.LifecycleState),
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func applyContactUpdates(c *entity.Contact, props map[string]interface{}) {
	if v, ok := props["first_name"].(string); ok {
		c.FirstName = v
	}
	if v, ok := props["last_name"].(string); ok {
		c.LastName = v
	}
	if v, ok := props["email"].(string); ok {
		c.Email = &v
	}
	if v, ok := props["phone"].(string); ok {
		c.Phone = &v
	}
	if v, ok := props["source"].(string); ok {
		c.Source = &v
	}

	custom := extractCustomProperties(props, []string{
		"first_name", "last_name", "email", "phone", "company_id", "source", "tags",
	})
	if len(custom) > 0 {
		var existing map[string]interface{}
		json.Unmarshal(c.CustomProperties, &existing)
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range custom {
			existing[k] = v
		}
		b, _ := json.Marshal(existing)
		c.CustomProperties = datatypes.JSON(b)
	}
}

func applyCompanyUpdates(c *entity.Company, props map[string]interface{}) {
	if v, ok := props["name"].(string); ok {
		c.Name = v
	}
	if v, ok := props["domain"].(string); ok {
		c.Domain = &v
	}
	if v, ok := props["industry"].(string); ok {
		c.Industry = &v
	}

	custom := extractCustomProperties(props, []string{
		"name", "domain", "industry", "employee_count", "annual_revenue", "address",
	})
	if len(custom) > 0 {
		var existing map[string]interface{}
		json.Unmarshal(c.CustomProperties, &existing)
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range custom {
			existing[k] = v
		}
		b, _ := json.Marshal(existing)
		c.CustomProperties = datatypes.JSON(b)
	}
}

func applyOpportunityUpdates(o *entity.Opportunity, props map[string]interface{}) {
	if v, ok := props["name"].(string); ok {
		o.Name = v
	}
	if v, ok := props["stage_id"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			o.StageID = id
		}
	}
	if v, ok := props["monetary_value"].(float64); ok {
		i := int64(v)
		o.MonetaryValue = &i
	}
	if v, ok := props["currency"].(string); ok {
		o.Currency = v
	}

	custom := extractCustomProperties(props, []string{
		"name", "pipeline_id", "stage_id", "contact_id", "company_id",
		"monetary_value", "currency", "expected_close_date", "assigned_to",
	})
	if len(custom) > 0 {
		var existing map[string]interface{}
		json.Unmarshal(o.CustomProperties, &existing)
		if existing == nil {
			existing = make(map[string]interface{})
		}
		for k, v := range custom {
			existing[k] = v
		}
		b, _ := json.Marshal(existing)
		o.CustomProperties = datatypes.JSON(b)
	}
}
