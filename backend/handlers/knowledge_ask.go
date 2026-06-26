package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"arke/backend/models"
	"arke/backend/services"
)

type KnowledgeAskHandler struct {
	service *services.KnowledgeAskService
}

func NewKnowledgeAskHandler(service *services.KnowledgeAskService) *KnowledgeAskHandler {
	return &KnowledgeAskHandler{service: service}
}

func (h *KnowledgeAskHandler) Ask(c *gin.Context) {
	var req models.KnowledgeAskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}

	result, err := h.service.Ask(req)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, result)
}
