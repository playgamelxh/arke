package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"arke/backend/config"
	"arke/backend/db"
	"arke/backend/handlers"
	"arke/backend/routes"
	"arke/backend/services"
)

func main() {
	cfg := config.Load()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.MaxMultipartMemory = 512 << 20 // 512MB

	gormDB, err := db.Init(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败：%v", err)
	}

	settingsService := services.NewSettingsService(gormDB, cfg)
	storage := services.NewRuntimeStorage(settingsService)

	mineruClient := services.NewMinerUClient(cfg.MinerUBaseURL, cfg.MinerUTimeout)
	bailianClient := services.NewBailianClient(cfg)

	// 知识库相关服务
	milvusBridgeURL := os.Getenv("MILVUS_BRIDGE_URL")
	if milvusBridgeURL == "" {
		milvusBridgeURL = "http://milvus-bridge:8088"
	}
	milvusBridge := services.NewMilvusBridge(milvusBridgeURL)
	embeddingClient := services.NewEmbeddingClient(cfg)
	if err := milvusBridge.Health(); err != nil {
		log.Printf("警告：Milvus Bridge 不可用，知识库检索功能将不可用：%v", err)
	}

	// 业务服务
	documentService := services.NewDocumentService(gormDB, cfg, storage, mineruClient, bailianClient)
	kbDocumentService := services.NewKBDocumentService(gormDB, cfg, storage, settingsService, mineruClient, milvusBridge, embeddingClient)
	kbService := services.NewKnowledgeBaseService(gormDB, milvusBridge, embeddingClient)
	qaService := services.NewQAService(gormDB, bailianClient)
	knowledgeAskService := services.NewKnowledgeAskService(kbService, bailianClient)

	// Handlers
	documentHandler := handlers.NewDocumentHandler(documentService)
	qaHandler := handlers.NewQAHandler(qaService)
	kbHandler := handlers.NewKnowledgeBaseHandler(kbService, kbDocumentService)
	kbDocHandler := handlers.NewKBDocumentHandler(kbDocumentService)
	knowledgeAskHandler := handlers.NewKnowledgeAskHandler(knowledgeAskService)
	settingsHandler := handlers.NewSettingsHandler(settingsService, storage)

	routes.RegisterRoutes(router, documentHandler, qaHandler, kbHandler, kbDocHandler, knowledgeAskHandler, settingsHandler, cfg.AllowedOrigins)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("服务启动成功，监听端口：%s\n", cfg.Port)
	fmt.Printf("健康检查地址：http://localhost:%s/health\n", cfg.Port)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务启动失败：%v", err)
	}
}
