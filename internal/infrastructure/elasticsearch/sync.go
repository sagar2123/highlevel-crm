package elasticsearch

import (
	"context"
	"log"

	"github.com/sagar2123/highlevel-crm/internal/domain/repository"
)

type SyncService struct {
	search repository.SearchRepository
}

func NewSyncService(search repository.SearchRepository) *SyncService {
	return &SyncService{search: search}
}

func (s *SyncService) IndexDocument(ctx context.Context, objectType string, id string, doc map[string]interface{}) {
	if err := s.search.Index(ctx, objectType, id, doc); err != nil {
		log.Printf("failed to index %s/%s: %v", objectType, id, err)
	}
}

func (s *SyncService) RemoveDocument(ctx context.Context, objectType string, id string) {
	if err := s.search.Remove(ctx, objectType, id); err != nil {
		log.Printf("failed to remove %s/%s from index: %v", objectType, id, err)
	}
}
