package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"arke/backend/services"
)

type SettingsHandler struct {
	settings *services.SettingsService
	storage  *services.RuntimeStorage
}

func NewSettingsHandler(settings *services.SettingsService, storage *services.RuntimeStorage) *SettingsHandler {
	return &SettingsHandler{settings: settings, storage: storage}
}

func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.settings.GetSettings()
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, settings)
}

func (h *SettingsHandler) Update(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "参数错误："+err.Error())
		return
	}
	settings, err := h.settings.UpdateSettings(req)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, settings)
}

func (h *SettingsHandler) TestRustFS(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		req = map[string]string{}
	}
	if err := h.storage.TestRustFS(req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, gin.H{"ok": true, "message": "RustFS 连接、读写、删除测试通过"})
}

func (h *SettingsHandler) TestParse(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		req = map[string]string{}
	}
	if err := h.settings.TestParseSettings(req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	ok(c, gin.H{"ok": true, "message": "文档解析配置检测通过"})
}
