package models

import "time"

type ApiResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type DocumentResponse struct {
	ID              uint           `json:"id"`
	KnowledgeBaseID *uint          `json:"knowledgeBaseId"`
	Name            string         `json:"name"`
	OriginalName    string         `json:"originalName"`
	FileType        string         `json:"fileType"`
	FileSize        int64          `json:"fileSize"`
	Status          DocumentStatus `json:"status"`
	ParseError      string         `json:"parseError"`
	ChunkStrategy   string         `json:"chunkStrategy"`
	ChunkSize       int            `json:"chunkSize"`
	ChunkOverlap    int            `json:"chunkOverlap"`
	SegmentCount    int64          `json:"segmentCount"`
	QACount         int64          `json:"qaCount"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type QAResponse struct {
	ID              uint      `json:"id"`
	DocumentID      uint      `json:"documentId"`
	DocumentName    string    `json:"documentName"`
	SourceSegmentID *uint     `json:"sourceSegmentId"`
	Question        string    `json:"question"`
	Answer          string    `json:"answer"`
	Tags            []string  `json:"tags"`
	Enabled         bool      `json:"enabled"`
	Confidence      float64   `json:"confidence"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type QAGenerateTaskResponse struct {
	ID              uint                 `json:"id"`
	KnowledgeBaseID uint                 `json:"knowledgeBaseId"`
	DocumentID      uint                 `json:"documentId"`
	Status          QAGenerateTaskStatus `json:"status"`
	Progress        int                  `json:"progress"`
	Message         string               `json:"message"`
	TargetCount     int                  `json:"targetCount"`
	GeneratedCount  int                  `json:"generatedCount"`
	CurrentBatch    int                  `json:"currentBatch"`
	TotalBatches    int                  `json:"totalBatches"`
	BatchSize       int                  `json:"batchSize"`
	Items           []GeneratedQAItem    `json:"items,omitempty"`
	Error           string               `json:"error,omitempty"`
	CreatedAt       time.Time            `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
}

type PageResponse[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

type GenerateAnswerResponse struct {
	Answer          string  `json:"answer"`
	SourceExcerpt   string  `json:"sourceExcerpt"`
	SourceSegmentID *uint   `json:"sourceSegmentId,omitempty"`
	Confidence      float64 `json:"confidence"`
}

type KnowledgeAskSource struct {
	DocumentID       uint    `json:"documentId"`
	SourceSegmentID  *uint   `json:"sourceSegmentId"`
	Content          string  `json:"content"`
	Score            float64 `json:"score"`
	OriginalDistance float64 `json:"originalDistance"`
}

type KnowledgeAskResponse struct {
	Answer        string               `json:"answer"`
	Confidence    float64              `json:"confidence"`
	Sources       []KnowledgeAskSource `json:"sources"`
	RecallCount   int                  `json:"recallCount"`
	UseCount      int                  `json:"useCount"`
	RerankMode    string               `json:"rerankMode"`
	SourceExcerpt string               `json:"sourceExcerpt"`
}

type StatsResponse struct {
	Documents       int64              `json:"documents"`
	Parsed          int64              `json:"parsed"`
	Failed          int64              `json:"failed"`
	QA              int64              `json:"qa"`
	RecentDocuments []DocumentResponse `json:"recentDocuments"`
}

type SettingsResponse struct {
	MaxFileSizeMB     string `json:"maxFileSizeMB"`
	AllowedTypes      string `json:"allowedTypes"`
	DefaultModel      string `json:"defaultModel"`
	DefaultDifficulty string `json:"defaultDifficulty"`
}
