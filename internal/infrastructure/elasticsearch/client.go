package elasticsearch

import (
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sagar2123/highlevel-crm/config"
)

func NewClient(cfg config.ES) (*elasticsearch.Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{cfg.URL},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %w", err)
	}

	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to ping elasticsearch: %w", err)
	}
	defer res.Body.Close()

	log.Println("connected to elasticsearch")
	return es, nil
}
