package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/sagar2123/highlevel-crm/config"
	"github.com/sagar2123/highlevel-crm/internal/application/crm"
	"github.com/sagar2123/highlevel-crm/internal/infrastructure/database"
	"github.com/sagar2123/highlevel-crm/internal/infrastructure/elasticsearch"
	router "github.com/sagar2123/highlevel-crm/internal/infrastructure/http"
)

func main() {
	cfg := config.Load()

	db, err := database.NewPostgresConnection(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	esClient, err := elasticsearch.NewClient(cfg.ES)
	if err != nil {
		log.Fatalf("failed to connect to elasticsearch: %v", err)
	}

	tenantDB := database.NewTenantDB(db)

	contactRepo := database.NewContactRepository(tenantDB)
	companyRepo := database.NewCompanyRepository(tenantDB)
	opportunityRepo := database.NewOpportunityRepository(tenantDB)
	pipelineRepo := database.NewPipelineRepository(tenantDB)
	schemaRepo := database.NewCustomObjectSchemaRepository(tenantDB)
	recordRepo := database.NewCustomObjectRecordRepository(tenantDB)
	assocDefRepo := database.NewAssociationDefinitionRepository(tenantDB)
	assocRepo := database.NewAssociationRepository(tenantDB)
	searchRepo := elasticsearch.NewSearchRepository(esClient)

	syncService := elasticsearch.NewSyncService(searchRepo)

	service := crm.NewService(
		contactRepo,
		companyRepo,
		opportunityRepo,
		pipelineRepo,
		schemaRepo,
		recordRepo,
		assocDefRepo,
		assocRepo,
		searchRepo,
		syncService,
	)

	ctrl := crm.NewController(service)
	r := router.NewRouter(ctrl)

	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("starting server on :%s", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	log.Println("server stopped")
}
