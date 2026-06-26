package models

import "time"

type KnowledgeBase struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	Name             string    `json:"name" gorm:"size:128;uniqueIndex"`
	Description      string    `json:"description" gorm:"type:text"`
	EmbeddingModel   string    `json:"embeddingModel" gorm:"size:128;default:text-embedding-v3"`
	EmbeddingDim     int       `json:"embeddingDim" gorm:"default:1024"`
	IndexType        string    `json:"indexType" gorm:"size:32;default:HNSW"`
	IndexParams      string    `json:"indexParams" gorm:"type:json"`
	ChunkStrategy    string    `json:"chunkStrategy" gorm:"size:32;default:paragraph"`
	ChunkSize        int       `json:"chunkSize" gorm:"default:500"`
	ChunkOverlap     int       `json:"chunkOverlap" gorm:"default:50"`
	MilvusCollection string    `json:"milvusCollection" gorm:"size:128;uniqueIndex"`
	DocCount         int       `json:"docCount" gorm:"default:0"`
	VectorCount      int64     `json:"vectorCount" gorm:"default:0"`
	Documents        []Document
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}
