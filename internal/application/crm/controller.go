package crm

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sagar2123/highlevel-crm/internal/domain/valueobject"
)

type Controller struct {
	service *Service
}

func NewController(service *Service) *Controller {
	return &Controller{service: service}
}

func (ctrl *Controller) CreateRecord(c *gin.Context) {
	objectType := c.Param("object_type")
	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	resp, err := ctrl.service.CreateRecord(c.Request.Context(), objectType, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "CREATE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) GetRecord(c *gin.Context) {
	objectType := c.Param("object_type")
	id, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	resp, err := ctrl.service.GetRecord(c.Request.Context(), objectType, id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: ErrorDetail{Code: "NOT_FOUND", Message: "record not found"}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) UpdateRecord(c *gin.Context) {
	objectType := c.Param("object_type")
	id, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	var req UpdateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	resp, err := ctrl.service.UpdateRecord(c.Request.Context(), objectType, id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "UPDATE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) DeleteRecord(c *gin.Context) {
	objectType := c.Param("object_type")
	id, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	if err := ctrl.service.DeleteRecord(c.Request.Context(), objectType, id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "DELETE_FAILED", Message: err.Error()}})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *Controller) SearchRecords(c *gin.Context) {
	objectType := c.Param("object_type")
	var req valueobject.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	result, err := ctrl.service.SearchRecords(c.Request.Context(), objectType, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "SEARCH_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (ctrl *Controller) ListRecords(c *gin.Context) {
	objectType := c.Param("object_type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := ctrl.service.ListRecords(c.Request.Context(), objectType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "LIST_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) ArchiveRecord(c *gin.Context) {
	objectType := c.Param("object_type")
	id, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	if err := ctrl.service.ArchiveRecord(c.Request.Context(), objectType, id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "ARCHIVE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "archived"})
}

func (ctrl *Controller) RestoreRecord(c *gin.Context) {
	objectType := c.Param("object_type")
	id, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	if err := ctrl.service.RestoreRecord(c.Request.Context(), objectType, id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "RESTORE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "restored"})
}

func (ctrl *Controller) CreateAssociation(c *gin.Context) {
	recordID, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	var req CreateAssociationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	objectType := c.Param("object_type")
	resp, err := ctrl.service.CreateAssociation(c.Request.Context(), objectType, recordID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "ASSOC_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) ListAssociations(c *gin.Context) {
	recordID, err := uuid.Parse(c.Param("record_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid record id"}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	assocs, total, err := ctrl.service.ListAssociations(c.Request.Context(), recordID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "LIST_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": assocs, "total": total, "page": page, "page_size": pageSize})
}

func (ctrl *Controller) DeleteAssociation(c *gin.Context) {
	assocID, err := uuid.Parse(c.Param("assoc_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid association id"}})
		return
	}

	if err := ctrl.service.DeleteAssociation(c.Request.Context(), assocID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "DELETE_FAILED", Message: err.Error()}})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *Controller) CreateSchema(c *gin.Context) {
	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	resp, err := ctrl.service.CreateSchema(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "CREATE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) ListSchemas(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	schemas, total, err := ctrl.service.ListSchemas(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "LIST_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": schemas, "total": total, "page": page, "page_size": pageSize})
}

func (ctrl *Controller) GetSchema(c *gin.Context) {
	id, err := uuid.Parse(c.Param("schema_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid schema id"}})
		return
	}

	resp, err := ctrl.service.GetSchema(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: ErrorDetail{Code: "NOT_FOUND", Message: "schema not found"}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) UpdateSchema(c *gin.Context) {
	id, err := uuid.Parse(c.Param("schema_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid schema id"}})
		return
	}

	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	resp, err := ctrl.service.UpdateSchema(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "UPDATE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) DeleteSchema(c *gin.Context) {
	id, err := uuid.Parse(c.Param("schema_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid schema id"}})
		return
	}

	if err := ctrl.service.DeleteSchema(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "DELETE_FAILED", Message: err.Error()}})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *Controller) CreateAssociationDefinition(c *gin.Context) {
	var req CreateAssociationDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	resp, err := ctrl.service.CreateAssociationDefinition(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "CREATE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) ListAssociationDefinitions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	defs, total, err := ctrl.service.ListAssociationDefinitions(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "LIST_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": defs, "total": total, "page": page, "page_size": pageSize})
}

func (ctrl *Controller) DeleteAssociationDefinition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("def_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid definition id"}})
		return
	}

	if err := ctrl.service.DeleteAssociationDefinition(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "DELETE_FAILED", Message: err.Error()}})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctrl *Controller) CreatePipeline(c *gin.Context) {
	var req PipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	resp, err := ctrl.service.CreatePipeline(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "CREATE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (ctrl *Controller) GetPipeline(c *gin.Context) {
	id, err := uuid.Parse(c.Param("pipeline_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid pipeline id"}})
		return
	}

	resp, err := ctrl.service.GetPipeline(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: ErrorDetail{Code: "NOT_FOUND", Message: "pipeline not found"}})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (ctrl *Controller) AddPipelineStage(c *gin.Context) {
	pipelineID, err := uuid.Parse(c.Param("pipeline_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "INVALID_ID", Message: "invalid pipeline id"}})
		return
	}

	var req StageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: ErrorDetail{Code: "VALIDATION_ERROR", Message: err.Error()}})
		return
	}

	if err := ctrl.service.AddPipelineStage(c.Request.Context(), pipelineID, req); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: ErrorDetail{Code: "STAGE_FAILED", Message: err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "stage added"})
}
