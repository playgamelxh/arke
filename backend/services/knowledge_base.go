package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"arke/backend/models"
)

type KnowledgeBaseService struct {
	db        *gorm.DB
	milvus    *MilvusBridge
	embedding *EmbeddingClient
}

func NewKnowledgeBaseService(db *gorm.DB, milvus *MilvusBridge, embedding *EmbeddingClient) *KnowledgeBaseService {
	return &KnowledgeBaseService{
		db:        db,
		milvus:    milvus,
		embedding: embedding,
	}
}

type CreateKnowledgeBaseRequest struct {
	Name           string         `json:"name" binding:"required"`
	Description    string         `json:"description"`
	EmbeddingModel string         `json:"embeddingModel"`
	EmbeddingDim   int            `json:"embeddingDim"`
	IndexType      string         `json:"indexType"`
	IndexParams    map[string]any `json:"indexParams"`
	ChunkStrategy  string         `json:"chunkStrategy"`
	ChunkSize      int            `json:"chunkSize"`
	ChunkOverlap   int            `json:"chunkOverlap"`
}

func (s *KnowledgeBaseService) Create(req CreateKnowledgeBaseRequest) (*models.KnowledgeBase, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("知识库名称不能为空")
	}

	embeddingModel := strings.TrimSpace(req.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = s.embedding.DefaultModel()
	}
	if !IsSupportedEmbeddingModel(embeddingModel) {
		return nil, fmt.Errorf("不支持的 embedding 模型：%s", req.EmbeddingModel)
	}

	embeddingDim := req.EmbeddingDim
	if embeddingDim <= 0 {
		embeddingDim = s.embedding.Dimension()
	}
	if !IsSupportedEmbeddingDim(embeddingModel, embeddingDim) {
		return nil, fmt.Errorf("模型 %s 不支持维度 %d", embeddingModel, embeddingDim)
	}

	indexType := models.IndexType(strings.ToUpper(req.IndexType))
	if indexType == "" {
		indexType = models.IndexHNSW
	}
	if !indexType.Valid() {
		return nil, fmt.Errorf("不支持的索引类型：%s", req.IndexType)
	}

	chunkStrategy := models.ChunkStrategy(req.ChunkStrategy)
	if chunkStrategy == "" {
		chunkStrategy = models.ChunkStrategyParagraph
	}
	if !chunkStrategy.Valid() {
		return nil, fmt.Errorf("不支持的切片策略：%s", req.ChunkStrategy)
	}

	params := req.IndexParams
	if len(params) == 0 {
		params = indexType.DefaultParams()
	}
	paramsJSON, _ := json.Marshal(params)

	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 500
	}
	chunkOverlap := req.ChunkOverlap
	if chunkOverlap < 0 {
		chunkOverlap = 50
	}

	// 生成 Milvus collection 名称
	collectionName := fmt.Sprintf("kb_%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:16])

	// 在 Milvus 中创建 collection
	err := s.milvus.CreateCollection(CreateCollectionRequest{
		CollectionName: collectionName,
		Dimension:      embeddingDim,
		IndexType:      string(indexType),
		IndexParams:    params,
		MetricType:     "COSINE",
	})
	if err != nil {
		return nil, fmt.Errorf("创建 Milvus collection 失败：%w", err)
	}

	kb := &models.KnowledgeBase{
		Name:             name,
		Description:      req.Description,
		EmbeddingModel:   embeddingModel,
		EmbeddingDim:     embeddingDim,
		IndexType:        string(indexType),
		IndexParams:      string(paramsJSON),
		ChunkStrategy:    string(chunkStrategy),
		ChunkSize:        chunkSize,
		ChunkOverlap:     chunkOverlap,
		MilvusCollection: collectionName,
		DocCount:         0,
		VectorCount:      0,
	}

	if err := s.db.Create(kb).Error; err != nil {
		// 回滚 Milvus collection
		s.milvus.DropCollection(collectionName)
		return nil, err
	}

	return kb, nil
}

func (s *KnowledgeBaseService) List() ([]models.KnowledgeBase, error) {
	var kbs []models.KnowledgeBase
	if err := s.db.Order("created_at DESC").Find(&kbs).Error; err != nil {
		return nil, err
	}
	return kbs, nil
}

func (s *KnowledgeBaseService) Get(id uint) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, id).Error; err != nil {
		return nil, err
	}
	return &kb, nil
}

type UpdateKnowledgeBaseRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (s *KnowledgeBaseService) Update(id uint, req UpdateKnowledgeBaseRequest) (*models.KnowledgeBase, error) {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) > 0 {
		if err := s.db.Model(&kb).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.Get(id)
}

type UpdateIndexTypeRequest struct {
	IndexType   string         `json:"indexType" binding:"required"`
	IndexParams map[string]any `json:"indexParams"`
}

func (s *KnowledgeBaseService) UpdateIndexType(id uint, req UpdateIndexTypeRequest) error {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, id).Error; err != nil {
		return err
	}

	if kb.DocCount > 0 || kb.VectorCount > 0 {
		return fmt.Errorf("知识库已有数据，无法更换索引类型。请先清空知识库中的文档")
	}

	indexType := models.IndexType(strings.ToUpper(req.IndexType))
	if !indexType.Valid() {
		return fmt.Errorf("不支持的索引类型：%s", req.IndexType)
	}

	params := req.IndexParams
	if len(params) == 0 {
		params = indexType.DefaultParams()
	}
	paramsJSON, _ := json.Marshal(params)

	// 重建 Milvus collection
	if err := s.milvus.DropCollection(kb.MilvusCollection); err != nil {
		return fmt.Errorf("删除旧 collection 失败：%w", err)
	}

	err := s.milvus.CreateCollection(CreateCollectionRequest{
		CollectionName: kb.MilvusCollection,
		Dimension:      kb.EmbeddingDim,
		IndexType:      string(indexType),
		IndexParams:    params,
		MetricType:     "COSINE",
	})
	if err != nil {
		return fmt.Errorf("创建新 collection 失败：%w", err)
	}

	return s.db.Model(&kb).Updates(map[string]any{
		"index_type":   string(indexType),
		"index_params": string(paramsJSON),
	}).Error
}

func (s *KnowledgeBaseService) Delete(id uint) error {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, id).Error; err != nil {
		return err
	}

	// 删除 Milvus collection
	if err := s.milvus.DropCollection(kb.MilvusCollection); err != nil {
		return fmt.Errorf("删除 Milvus collection 失败：%w", err)
	}

	// 删除 MySQL 记录（级联删除 documents 和 segments）
	return s.db.Delete(&kb).Error
}

// Search 检索知识库
func (s *KnowledgeBaseService) Search(kbID uint, query string, topK int) ([]SearchHit, error) {
	var kb models.KnowledgeBase
	if err := s.db.First(&kb, kbID).Error; err != nil {
		return nil, err
	}

	if topK <= 0 {
		topK = 5
	}
	if topK > 50 {
		topK = 50
	}

	vector, err := s.embedding.EmbedWith(query, kb.EmbeddingModel, kb.EmbeddingDim)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败：%w", err)
	}

	resp, err := s.milvus.Search(kb.MilvusCollection, SearchRequest{
		Vectors:      [][]float32{vector},
		TopK:         topK,
		OutputFields: []string{"id", "content", "doc_id", "segment_id", "kb_id"},
	})
	if err != nil {
		return nil, fmt.Errorf("向量检索失败：%w", err)
	}

	if len(resp.Results) == 0 {
		return []SearchHit{}, nil
	}
	hits := resp.Results[0]
	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Distance < hits[j].Distance
	})
	return hits, nil
}
