package models

import "time"

type Document struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	KnowledgeBaseID *uint          `json:"knowledgeBaseId" gorm:"index"`
	KnowledgeBase   *KnowledgeBase `json:"knowledgeBase,omitempty" gorm:"foreignKey:KnowledgeBaseID"`
	Name            string         `json:"name"`
	OriginalName    string         `json:"originalName"`
	FileType        string         `json:"fileType"`
	FileSize        int64          `json:"fileSize"`
	FilePath        string         `json:"-"`
	Status          DocumentStatus `json:"status"`
	ParseError      string         `json:"parseError"`
	ChunkStrategy   string         `json:"chunkStrategy" gorm:"size:32;default:paragraph"`
	ChunkSize       int            `json:"chunkSize" gorm:"default:500"`
	ChunkOverlap    int            `json:"chunkOverlap" gorm:"default:50"`
	Segments        []DocumentSegment
	QAs             []QAItem
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type DocumentSegment struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	DocumentID   uint       `json:"documentId"`
	VectorID     string     `json:"vectorId" gorm:"size:64;index"`
	SegmentType  string     `json:"segmentType"`
	SegmentIndex int        `json:"segmentIndex"`
	Title        string     `json:"title"`
	Content      string     `json:"content" gorm:"type:longtext"`
	IndexedAt    *time.Time `json:"indexedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
}
