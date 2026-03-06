package repository

import (
	"context"

	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type SearchRepository interface {
	Search(ctx context.Context, objectType string, req valueobject.SearchRequest) (*valueobject.SearchResult, error)
	Index(ctx context.Context, objectType string, id string, doc map[string]interface{}) error
	Remove(ctx context.Context, objectType string, id string) error
}
