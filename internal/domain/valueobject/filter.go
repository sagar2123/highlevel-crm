package valueobject

type SearchRequest struct {
	Filters  []FilterGroup `json:"filters"`
	Sort     []SortClause  `json:"sort,omitempty"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Query    string        `json:"query,omitempty"`
}

type FilterGroup struct {
	Operator   string            `json:"operator"`
	Conditions []FilterCondition `json:"conditions"`
}

type FilterCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type SortClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type SearchResult struct {
	Results []map[string]interface{} `json:"results"`
	Total   int64                    `json:"total"`
	Page    int                      `json:"page"`
	PageSize int                     `json:"page_size"`
	HasMore bool                     `json:"has_more"`
}

func (s *SearchRequest) Normalize() {
	if s.Page < 1 {
		s.Page = 1
	}
	if s.PageSize < 1 || s.PageSize > 100 {
		s.PageSize = 20
	}
}
