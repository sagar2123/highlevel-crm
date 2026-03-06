package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type searchRepo struct {
	client *elasticsearch.Client
}

func NewSearchRepository(client *elasticsearch.Client) *searchRepo {
	return &searchRepo{client: client}
}

func (r *searchRepo) Search(ctx context.Context, objectType string, req valueobject.SearchRequest) (*valueobject.SearchResult, error) {
	req.Normalize()

	tenantID, _ := ctx.Value("tenant_id").(string)
	query := buildESQuery(req, tenantID)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(objectType),
		r.client.Search.WithBody(&buf),
		r.client.Search.WithRouting(tenantID),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search returned error: %s", res.String())
	}

	var esResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(esResponse.Hits.Hits))
	for _, hit := range esResponse.Hits.Hits {
		results = append(results, hit.Source)
	}

	return &valueobject.SearchResult{
		Results:  results,
		Total:    esResponse.Hits.Total.Value,
		Page:     req.Page,
		PageSize: req.PageSize,
		HasMore:  int64(req.Page*req.PageSize) < esResponse.Hits.Total.Value,
	}, nil
}

func (r *searchRepo) Index(ctx context.Context, objectType string, id string, doc map[string]interface{}) error {
	tenantID, _ := ctx.Value("tenant_id").(string)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return fmt.Errorf("failed to encode document: %w", err)
	}

	res, err := r.client.Index(
		objectType,
		&buf,
		r.client.Index.WithContext(ctx),
		r.client.Index.WithDocumentID(id),
		r.client.Index.WithRouting(tenantID),
	)
	if err != nil {
		return fmt.Errorf("index request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index returned error: %s", res.String())
	}
	return nil
}

func (r *searchRepo) Remove(ctx context.Context, objectType string, id string) error {
	tenantID, _ := ctx.Value("tenant_id").(string)

	res, err := r.client.Delete(
		objectType,
		id,
		r.client.Delete.WithContext(ctx),
		r.client.Delete.WithRouting(tenantID),
	)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer res.Body.Close()
	return nil
}

func buildESQuery(req valueobject.SearchRequest, tenantID string) map[string]interface{} {
	must := []map[string]interface{}{
		{"term": map[string]interface{}{"tenant_id": tenantID}},
	}

	if req.Query != "" {
		must = append(must, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  req.Query,
				"fields": []string{"first_name", "last_name", "email", "name", "full_name"},
				"type":   "best_fields",
			},
		})
	}

	for _, group := range req.Filters {
		clauses := make([]map[string]interface{}, 0, len(group.Conditions))
		for _, cond := range group.Conditions {
			clause := buildCondition(cond)
			if clause != nil {
				clauses = append(clauses, clause)
			}
		}

		if len(clauses) == 0 {
			continue
		}

		if group.Operator == "OR" {
			must = append(must, map[string]interface{}{
				"bool": map[string]interface{}{"should": clauses, "minimum_should_match": 1},
			})
		} else {
			must = append(must, clauses...)
		}
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{"must": must},
		},
		"from": (req.Page - 1) * req.PageSize,
		"size": req.PageSize,
	}

	if len(req.Sort) > 0 {
		sorts := make([]map[string]interface{}, 0, len(req.Sort))
		for _, s := range req.Sort {
			sorts = append(sorts, map[string]interface{}{
				s.Field: map[string]interface{}{"order": s.Direction},
			})
		}
		query["sort"] = sorts
	}

	return query
}

func buildCondition(cond valueobject.FilterCondition) map[string]interface{} {
	switch cond.Operator {
	case "eq":
		return map[string]interface{}{"term": map[string]interface{}{cond.Field: cond.Value}}
	case "neq":
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": []map[string]interface{}{
					{"term": map[string]interface{}{cond.Field: cond.Value}},
				},
			},
		}
	case "contains":
		return map[string]interface{}{"match": map[string]interface{}{cond.Field: cond.Value}}
	case "in":
		return map[string]interface{}{"terms": map[string]interface{}{cond.Field: cond.Value}}
	case "gt":
		return map[string]interface{}{"range": map[string]interface{}{cond.Field: map[string]interface{}{"gt": cond.Value}}}
	case "gte":
		return map[string]interface{}{"range": map[string]interface{}{cond.Field: map[string]interface{}{"gte": cond.Value}}}
	case "lt":
		return map[string]interface{}{"range": map[string]interface{}{cond.Field: map[string]interface{}{"lt": cond.Value}}}
	case "lte":
		return map[string]interface{}{"range": map[string]interface{}{cond.Field: map[string]interface{}{"lte": cond.Value}}}
	case "between":
		vals, ok := cond.Value.([]interface{})
		if ok && len(vals) == 2 {
			return map[string]interface{}{"range": map[string]interface{}{cond.Field: map[string]interface{}{"gte": vals[0], "lte": vals[1]}}}
		}
	}
	return nil
}
