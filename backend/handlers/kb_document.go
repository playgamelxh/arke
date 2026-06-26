package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"arke/backend/models"
	"arke/backend/services"
)

type KBDocumentHandler struct {
	documentService *services.KBDocumentService
}

func NewKBDocumentHandler(doc *services.KBDocumentService) *KBDocumentHandler {
	return &KBDocumentHandler{documentService: doc}
}

func (h *KBDocumentHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}
	if file.Size > 500*1024*1024 {
		fail(c, http.StatusBadRequest, "文件大小不能超过 500MB")
		return
	}

	kbIDStr := c.PostForm("knowledgeBaseId")
	kbID, err := strconv.ParseUint(kbIDStr, 10, 64)
	if err != nil || kbID == 0 {
		fail(c, http.StatusBadRequest, "请指定 knowledgeBaseId")
		return
	}
	chunkStrategy := c.PostForm("chunkStrategy")
	chunkSize, _ := strconv.Atoi(c.PostForm("chunkSize"))
	chunkOverlap, _ := strconv.Atoi(c.PostForm("chunkOverlap"))

	doc, err := h.documentService.UploadDocument(uint(kbID), file, chunkStrategy, chunkSize, chunkOverlap)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, doc)
}

func (h *KBDocumentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	kbID, _ := strconv.ParseUint(c.Query("kbId"), 10, 64)

	docs, total, err := h.documentService.List(page, pageSize, uint(kbID))
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

func (h *KBDocumentHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	doc, err := h.documentService.GetDocument(uint(id))
	if err != nil {
		fail(c, http.StatusNotFound, "文档不存在")
		return
	}
	ok(c, doc)
}

func (h *KBDocumentHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	var req models.DocumentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	doc, err := h.documentService.UpdateDocument(uint(id), req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, doc)
}

func (h *KBDocumentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	if err := h.documentService.DeleteDocument(uint(id)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"status": "deleted"})
}

func (h *KBDocumentHandler) Parse(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	doc, err := h.documentService.GetDocument(uint(id))
	if err != nil {
		fail(c, http.StatusNotFound, "文档不存在")
		return
	}
	if err := h.documentService.UpdateDocumentStatus(uint(id), models.StatusParsing, ""); err != nil {
		fail(c, http.StatusInternalServerError, "更新状态失败")
		return
	}
	go func() {
		if err := h.documentService.ParseAndIndexDocument(doc); err != nil {
			h.documentService.UpdateDocumentStatus(uint(id), models.StatusFailed, err.Error())
		}
	}()
	ok(c, gin.H{"status": "parsing"})
}

func (h *KBDocumentHandler) Index(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	go func() {
		if err := h.documentService.IndexDocument(uint(id)); err != nil {
			h.documentService.UpdateDocumentStatus(uint(id), models.StatusFailed, err.Error())
		}
	}()
	ok(c, gin.H{"status": "indexing"})
}

func (h *KBDocumentHandler) GetSegments(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	segments, err := h.documentService.GetSegments(uint(id))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, segments)
}

func (h *KBDocumentHandler) UpdateSegment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 ID")
		return
	}
	segID, err := strconv.ParseUint(c.Param("segmentId"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 segmentId")
		return
	}
	var req models.SegmentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	if err := h.documentService.UpdateSegment(uint(segID), req.Title, req.Content); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	segments, err := h.documentService.GetSegments(uint(id))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	for _, segment := range segments {
		if segment.ID == uint(segID) {
			ok(c, segment)
			return
		}
	}
	fail(c, http.StatusNotFound, "切片不存在")
}

func (h *KBDocumentHandler) DeleteSegment(c *gin.Context) {
	segID, err := strconv.ParseUint(c.Param("segmentId"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的 segmentId")
		return
	}
	if err := h.documentService.DeleteSegment(uint(segID)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, gin.H{"status": "deleted"})
}
