package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"arke/backend/models"
	"arke/backend/services"
)

type DocumentHandler struct {
	documentService *services.DocumentService
}

func NewDocumentHandler(docService *services.DocumentService) *DocumentHandler {
	return &DocumentHandler{documentService: docService}
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, models.ApiResponse[any]{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, models.ApiResponse[any]{
		Code:    1,
		Message: message,
		Data:    nil,
	})
}

func (h *DocumentHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}

	if file.Size > 500*1024*1024 {
		fail(c, http.StatusBadRequest, "文件大小不能超过 500MB")
		return
	}

	doc, err := h.documentService.UploadDocument(file)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"id": doc.ID, "name": doc.Name})
}

func (h *DocumentHandler) Parse(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的文档ID")
		return
	}

	doc, err := h.documentService.GetDocumentByID(uint(id))
	if err != nil {
		fail(c, http.StatusNotFound, "文档不存在")
		return
	}

	if doc.Status == models.StatusParsing {
		fail(c, http.StatusConflict, "文档正在解析中")
		return
	}

	if err := h.documentService.ParseDocument(doc); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"message": "解析完成"})
}

func (h *DocumentHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	list, total, err := h.documentService.GetDocuments(page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, models.PageResponse[models.DocumentResponse]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *DocumentHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的文档ID")
		return
	}

	doc, err := h.documentService.GetDocumentByID(uint(id))
	if err != nil {
		fail(c, http.StatusNotFound, "文档不存在")
		return
	}

	ok(c, doc)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的文档ID")
		return
	}

	if err := h.documentService.DeleteDocument(uint(id)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"deleted": true})
}

func (h *DocumentHandler) GetSegments(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, err := strconv.ParseUint(docIDStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的文档ID")
		return
	}

	segments, err := h.documentService.GetDocumentSegments(uint(docID))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, segments)
}

func (h *DocumentHandler) UpdateSegment(c *gin.Context) {
	idStr := c.Param("segmentId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的段落ID")
		return
	}

	var req models.SegmentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	if err := h.documentService.UpdateSegment(uint(id), req.Title, req.Content); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"message": "更新成功"})
}

func (h *DocumentHandler) DeleteSegment(c *gin.Context) {
	idStr := c.Param("segmentId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的段落ID")
		return
	}

	if err := h.documentService.DeleteSegment(uint(id)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"message": "删除成功"})
}

func (h *DocumentHandler) Stats(c *gin.Context) {
	stats, err := h.documentService.GetStats()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, stats)
}
