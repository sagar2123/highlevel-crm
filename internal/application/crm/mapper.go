package crm

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sagar2123/highlevel-crm/internal/domain/entity"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
	"gorm.io/datatypes"
)

func contactFromProperties(props map[string]interface{}, tenantID string) *entity.Contact {
	c := &entity.Contact{}
	tID, _ := uuid.Parse(tenantID)
	c.TenantID = tID
	c.LifecycleState = valueobject.LifecycleActive

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
	if v, ok := props["company_id"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			c.CompanyID = &id
		}
	}
	if v, ok := props["source"].(string); ok {
		c.Source = &v
	}
	if v, ok := props["tags"].([]interface{}); ok {
		tags := make(pq.StringArray, 0, len(v))
		for _, t := range v {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
		c.Tags = tags
	}

	custom := extractCustomProperties(props, []string{
		"first_name", "last_name", "email", "phone", "company_id", "source", "tags",
	})
	if len(custom) > 0 {
		b, _ := json.Marshal(custom)
		c.CustomProperties = datatypes.JSON(b)
	}
	return c
}

func contactToProperties(c *entity.Contact) map[string]interface{} {
	props := map[string]interface{}{
		"first_name": c.FirstName,
		"last_name":  c.LastName,
	}
	if c.Email != nil {
		props["email"] = *c.Email
	}
	if c.Phone != nil {
		props["phone"] = *c.Phone
	}
	if c.CompanyID != nil {
		props["company_id"] = c.CompanyID.String()
	}
	if c.Source != nil {
		props["source"] = *c.Source
	}
	if len(c.Tags) > 0 {
		props["tags"] = []string(c.Tags)
	}

	var custom map[string]interface{}
	if len(c.CustomProperties) > 0 {
		json.Unmarshal(c.CustomProperties, &custom)
		for k, v := range custom {
			props[k] = v
		}
	}
	return props
}

func companyFromProperties(props map[string]interface{}, tenantID string) *entity.Company {
	c := &entity.Company{}
	tID, _ := uuid.Parse(tenantID)
	c.TenantID = tID
	c.LifecycleState = valueobject.LifecycleActive

	if v, ok := props["name"].(string); ok {
		c.Name = v
	}
	if v, ok := props["domain"].(string); ok {
		c.Domain = &v
	}
	if v, ok := props["industry"].(string); ok {
		c.Industry = &v
	}
	if v, ok := props["employee_count"].(float64); ok {
		i := int(v)
		c.EmployeeCount = &i
	}
	if v, ok := props["annual_revenue"].(float64); ok {
		i := int64(v)
		c.AnnualRevenue = &i
	}

	custom := extractCustomProperties(props, []string{
		"name", "domain", "industry", "employee_count", "annual_revenue", "address",
	})
	if len(custom) > 0 {
		b, _ := json.Marshal(custom)
		c.CustomProperties = datatypes.JSON(b)
	}
	return c
}

func companyToProperties(c *entity.Company) map[string]interface{} {
	props := map[string]interface{}{
		"name": c.Name,
	}
	if c.Domain != nil {
		props["domain"] = *c.Domain
	}
	if c.Industry != nil {
		props["industry"] = *c.Industry
	}
	if c.EmployeeCount != nil {
		props["employee_count"] = *c.EmployeeCount
	}
	if c.AnnualRevenue != nil {
		props["annual_revenue"] = *c.AnnualRevenue
	}

	var custom map[string]interface{}
	if len(c.CustomProperties) > 0 {
		json.Unmarshal(c.CustomProperties, &custom)
		for k, v := range custom {
			props[k] = v
		}
	}
	return props
}

func opportunityFromProperties(props map[string]interface{}, tenantID string) *entity.Opportunity {
	o := &entity.Opportunity{}
	tID, _ := uuid.Parse(tenantID)
	o.TenantID = tID
	o.LifecycleState = valueobject.LifecycleActive
	o.Currency = "USD"

	if v, ok := props["name"].(string); ok {
		o.Name = v
	}
	if v, ok := props["pipeline_id"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			o.PipelineID = id
		}
	}
	if v, ok := props["stage_id"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			o.StageID = id
		}
	}
	if v, ok := props["contact_id"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			o.ContactID = &id
		}
	}
	if v, ok := props["company_id"].(string); ok {
		if id, err := uuid.Parse(v); err == nil {
			o.CompanyID = &id
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
		b, _ := json.Marshal(custom)
		o.CustomProperties = datatypes.JSON(b)
	}
	return o
}

func opportunityToProperties(o *entity.Opportunity) map[string]interface{} {
	props := map[string]interface{}{
		"name":        o.Name,
		"pipeline_id": o.PipelineID.String(),
		"stage_id":    o.StageID.String(),
		"currency":    o.Currency,
	}
	if o.ContactID != nil {
		props["contact_id"] = o.ContactID.String()
	}
	if o.CompanyID != nil {
		props["company_id"] = o.CompanyID.String()
	}
	if o.MonetaryValue != nil {
		props["monetary_value"] = *o.MonetaryValue
	}
	if o.ExpectedCloseDate != nil {
		props["expected_close_date"] = o.ExpectedCloseDate
	}
	if o.AssignedTo != nil {
		props["assigned_to"] = o.AssignedTo.String()
	}

	var custom map[string]interface{}
	if len(o.CustomProperties) > 0 {
		json.Unmarshal(o.CustomProperties, &custom)
		for k, v := range custom {
			props[k] = v
		}
	}
	return props
}

func extractCustomProperties(props map[string]interface{}, builtInKeys []string) map[string]interface{} {
	known := make(map[string]bool, len(builtInKeys))
	for _, k := range builtInKeys {
		known[k] = true
	}
	custom := make(map[string]interface{})
	for k, v := range props {
		if !known[k] {
			custom[k] = v
		}
	}
	return custom
}
