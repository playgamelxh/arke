package routes

import (
	"github.com/gin-gonic/gin"

	"arke/backend/handlers"
	"arke/backend/middleware"
)

func RegisterRoutes(
	router *gin.Engine,
	docHandler *handlers.DocumentHandler,
	qaHandler *handlers.QAHandler,
	kbHandler *handlers.KnowledgeBaseHandler,
	kbDocHandler *handlers.KBDocumentHandler,
	knowledgeAskHandler *handlers.KnowledgeAskHandler,
	settingsHandler *handlers.SettingsHandler,
	allowedOrigins []string,
) {
	router.Use(middleware.CorsMiddleware(allowedOrigins))

	api := router.Group("/api")
	{
		api.GET("/stats", docHandler.Stats)
		api.GET("/settings", settingsHandler.Get)
		api.PUT("/settings", settingsHandler.Update)
		api.POST("/settings/test-rustfs", settingsHandler.TestRustFS)
		api.POST("/settings/test-parse", settingsHandler.TestParse)

		// 知识库管理
		kbs := api.Group("/knowledge-bases")
		{
			kbs.GET("", kbHandler.List)
			kbs.GET("/embedding-models", kbHandler.EmbeddingModels)
			kbs.POST("", kbHandler.Create)
			kbs.GET("/:id", kbHandler.Get)
			kbs.PUT("/:id", kbHandler.Update)
			kbs.DELETE("/:id", kbHandler.Delete)
			kbs.PUT("/:id/index", kbHandler.UpdateIndexType)
			kbs.GET("/:id/documents", kbHandler.ListDocuments)
			kbs.POST("/search", kbHandler.Search)
		}

		// 文档（按知识库维度）
		documents := api.Group("/documents")
		{
			documents.POST("/upload", kbDocHandler.Upload)
			documents.GET("", kbDocHandler.List)

			documents.POST("/:id/parse", kbDocHandler.Parse)
			documents.POST("/:id/index", kbDocHandler.Index)
			documents.GET("/:id", kbDocHandler.Get)
			documents.PUT("/:id", kbDocHandler.Update)
			documents.DELETE("/:id", kbDocHandler.Delete)
			documents.GET("/:id/segments", kbDocHandler.GetSegments)

			documents.PUT("/:id/segments/:segmentId", kbDocHandler.UpdateSegment)
			documents.DELETE("/:id/segments/:segmentId", kbDocHandler.DeleteSegment)
		}

		ask := api.Group("/knowledge-ask")
		{
			ask.POST("", knowledgeAskHandler.Ask)
		}

		qa := api.Group("/qa")
		{
			qa.POST("/generate-preview", qaHandler.GenerateQA)
			qa.GET("/generate-tasks/:id", qaHandler.GetQATask)
			qa.POST("/save-generated", qaHandler.SaveGenerated)
			qa.POST("/generate-answer", qaHandler.GenerateAnswer)

			qa.GET("", qaHandler.List)
			qa.POST("", qaHandler.Upsert)

			qa.GET("/:id", qaHandler.Get)
			qa.PUT("/:id", qaHandler.Update)
			qa.PATCH("/:id/status", qaHandler.UpdateStatus)
			qa.DELETE("/:id", qaHandler.Delete)

			qa.DELETE("/batch", qaHandler.BatchDelete)
		}
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
