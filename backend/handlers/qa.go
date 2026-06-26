package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"arke/backend/models"
	"arke/backend/services"
)

type QAHandler struct {
	qaService *services.QAService
}

func NewQAHandler(qaService *services.QAService) *QAHandler {
	return &QAHandler{qaService: qaService}
}

func (h *QAHandler) GenerateQA(c *gin.Context) {
	var req models.GenerateQARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	if req.Count <= 0 {
		req.Count = 10
	}
	if req.Count > 100 {
		req.Count = 100
	}

	task, err := h.qaService.GenerateQA(req.KnowledgeBaseID, req.Count, req.Difficulty, req.Instruction, req.Overwrite)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, models.QAGenerateTaskResponse{
		ID:              task.ID,
		KnowledgeBaseID: task.KnowledgeBaseID,
		DocumentID:      task.DocumentID,
		Status:          task.Status,
		Progress:        task.Progress,
		Message:         task.Message,
		TargetCount:     task.TaskCount,
		GeneratedCount:  0,
		CurrentBatch:    0,
		TotalBatches:    0,
		BatchSize:       0,
		Items:           nil,
		Error:           task.ErrorMessage,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	})
}

func (h *QAHandler) GetQATask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的任务ID")
		return
	}

	task, err := h.qaService.GetQATask(uint(taskID))
	if err != nil {
		fail(c, http.StatusNotFound, "任务不存在")
		return
	}

	ok(c, task)
}

func (h *QAHandler) List(c *gin.Context) {
	docIDStr := c.Query("documentId")
	var docID uint
	if docIDStr != "" {
		id, err := strconv.ParseUint(docIDStr, 10, 64)
		if err == nil {
			docID = uint(id)
		}
	}

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

	list, total, err := h.qaService.GetQAItems(docID, page, pageSize)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, models.PageResponse[models.QAResponse]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *QAHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的ID")
		return
	}

	item, err := h.qaService.GetQAItem(uint(id))
	if err != nil {
		fail(c, http.StatusNotFound, "问答不存在")
		return
	}

	ok(c, item)
}

func (h *QAHandler) Upsert(c *gin.Context) {
	var req models.QAUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	item := models.QAItem{
		ID:              0,
		DocumentID:      req.DocumentID,
		SourceSegmentID: req.SourceSegmentID,
		Question:        req.Question,
		Answer:          req.Answer,
		Tags:            "",
		Enabled:         req.Enabled,
		Confidence:      1.0,
	}

	result, err := h.qaService.UpsertQA(item)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, result)
}

func (h *QAHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var req models.QAUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	item := models.QAItem{
		ID:              uint(id),
		DocumentID:      req.DocumentID,
		SourceSegmentID: req.SourceSegmentID,
		Question:        req.Question,
		Answer:          req.Answer,
		Tags:            "",
		Enabled:         req.Enabled,
		Confidence:      1.0,
	}

	result, err := h.qaService.UpsertQA(item)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, result)
}

func (h *QAHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var req models.StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	if err := h.qaService.UpdateQAStatus(uint(id), req.Enabled); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"message": "更新成功"})
}

func (h *QAHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.qaService.DeleteQA(uint(id)); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"deleted": true})
}

func (h *QAHandler) BatchDelete(c *gin.Context) {
	var req models.BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	if len(req.IDs) == 0 {
		fail(c, http.StatusBadRequest, "请选择要删除的项目")
		return
	}

	if err := h.qaService.BatchDeleteQA(req.IDs); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"deleted": len(req.IDs)})
}

func (h *QAHandler) SaveGenerated(c *gin.Context) {
	var req models.SaveGeneratedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	if err := h.qaService.SaveGeneratedQA(req.KnowledgeBaseID, req.Items, req.Overwrite); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, gin.H{"saved": len(req.Items)})
}

func (h *QAHandler) GenerateAnswer(c *gin.Context) {
	var req models.GenerateAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, fmt.Sprintf("请求参数错误：%v", err))
		return
	}

	response, err := h.qaService.GenerateAnswer(req.DocumentID, req.Question)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}

	ok(c, response)
}
