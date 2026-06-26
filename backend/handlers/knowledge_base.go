package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"arke/backend/models"
	"arke/backend/services"
)

type KnowledgeBaseHandler struct {
	kbService      *services.KnowledgeBaseService
	documentService *services.KBDocumentService
}

func NewKnowledgeBaseHandler(kb *services.KnowledgeBaseService, doc *services.KBDocumentService) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		kbService:      kb,
		documentService: doc,
	}
}

func (h *KnowledgeBaseHandler) List(c *gin.Context) {
	kbs, err := h.kbService.List()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, kbs)
}

func (h *KnowledgeBaseHandler) EmbeddingModels(c *gin.Context) {
	models := services.SupportedEmbeddingModels()
	list := make([]gin.H, 0, len(models))
	for _, m := range models {
		list = append(list, gin.H{
			"model":      m,
			"dimensions": services.SupportedEmbeddingDims(m),
		})
	}
	ok(c, list)
}

func (h *KnowledgeBaseHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	kb, err := h.kbService.Get(uint(id))
	if err != nil {
		fail(c, http.StatusNotFound, "知识库不存在")
		return
	}
	ok(c, kb)
}

func (h *KnowledgeBaseHandler) Create(c *gin.Context) {
	var req models.KnowledgeBaseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	kb, err := h.kbService.Create(services.CreateKnowledgeBaseRequest{
		Name:          req.Name,
		Description:   req.Description,
		EmbeddingModel: req.EmbeddingModel,
		EmbeddingDim:  req.EmbeddingDim,
		IndexType:     req.IndexType,
		IndexParams:   req.IndexParams,
		ChunkStrategy: req.ChunkStrategy,
		ChunkSize:     req.ChunkSize,
		ChunkOverlap:  req.ChunkOverlap,
	})
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, kb)
}

func (h *KnowledgeBaseHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req models.KnowledgeBaseUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	kb, err := h.kbService.Update(uint(id), services.UpdateKnowledgeBaseRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, kb)
}

func (h *KnowledgeBaseHandler) UpdateIndexType(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req models.KnowledgeBaseIndexRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	if err := h.kbService.UpdateIndexType(uint(id), services.UpdateIndexTypeRequest{
		IndexType:   req.IndexType,
		IndexParams: req.IndexParams,
	}); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, gin.H{"status": "updated"})
}

func (h *KnowledgeBaseHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := h.kbService.Delete(uint(id)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"status": "deleted"})
}

func (h *KnowledgeBaseHandler) Search(c *gin.Context) {
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	results, err := h.kbService.Search(req.KBID, req.Query, req.TopK)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, results)
}

func (h *KnowledgeBaseHandler) ListDocuments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}

	docs, total, err := h.documentService.ListByKB(uint(id), page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, models.PageResponse[models.DocumentResponse]{
		List:     docs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}
