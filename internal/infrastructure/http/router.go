package http

import (
	"github.com/gin-gonic/gin"
	"github.com/sagar2123/highlevel-crm/internal/application/crm"
	"github.com/sagar2123/highlevel-crm/internal/infrastructure/middleware"
)

func NewRouter(ctrl *crm.Controller) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.ErrorHandler())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/crm")
	api.Use(middleware.TenantExtractor())

	objects := api.Group("/objects/:object_type")
	{
		objects.POST("", ctrl.CreateRecord)
		objects.GET("", ctrl.ListRecords)
		objects.GET("/:record_id", ctrl.GetRecord)
		objects.PATCH("/:record_id", ctrl.UpdateRecord)
		objects.DELETE("/:record_id", ctrl.DeleteRecord)
		objects.POST("/:record_id/search", ctrl.SearchRecords)
		objects.PATCH("/:record_id/archive", ctrl.ArchiveRecord)
		objects.PATCH("/:record_id/restore", ctrl.RestoreRecord)

		objects.POST("/:record_id/associations", ctrl.CreateAssociation)
		objects.GET("/:record_id/associations", ctrl.ListAssociations)
		objects.DELETE("/:record_id/associations/:assoc_id", ctrl.DeleteAssociation)
	}

	search := api.Group("/objects/:object_type")
	{
		search.POST("/search", ctrl.SearchRecords)
	}

	schemas := api.Group("/schemas")
	{
		schemas.POST("", ctrl.CreateSchema)
		schemas.GET("", ctrl.ListSchemas)
		schemas.GET("/:schema_id", ctrl.GetSchema)
		schemas.PATCH("/:schema_id", ctrl.UpdateSchema)
		schemas.DELETE("/:schema_id", ctrl.DeleteSchema)
	}

	assocDefs := api.Group("/association-definitions")
	{
		assocDefs.POST("", ctrl.CreateAssociationDefinition)
		assocDefs.GET("", ctrl.ListAssociationDefinitions)
		assocDefs.DELETE("/:def_id", ctrl.DeleteAssociationDefinition)
	}

	pipelines := api.Group("/pipelines")
	{
		pipelines.POST("", ctrl.CreatePipeline)
		pipelines.GET("", ctrl.ListRecords)
		pipelines.GET("/:pipeline_id", ctrl.GetPipeline)
		pipelines.POST("/:pipeline_id/stages", ctrl.AddPipelineStage)
	}

	return r
}
