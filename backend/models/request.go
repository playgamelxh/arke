package models

type GenerateQARequest struct {
	KnowledgeBaseID uint   `json:"knowledgeBaseId" binding:"required"`
	Count           int    `json:"count"`
	Difficulty      string `json:"difficulty"`
	Instruction     string `json:"instruction"`
	Overwrite       bool   `json:"overwrite"`
}

type SaveGeneratedRequest struct {
	KnowledgeBaseID uint              `json:"knowledgeBaseId" binding:"required"`
	Items           []GeneratedQAItem `json:"items" binding:"required"`
	Overwrite       bool              `json:"overwrite"`
}

type GeneratedQAItem struct {
	Question        string   `json:"question"`
	Answer          string   `json:"answer"`
	Keywords        []string `json:"keywords"`
	DocumentID      uint     `json:"documentId"`
	SourceSegmentID *uint    `json:"sourceSegmentId"`
	SourceExcerpt   string   `json:"sourceExcerpt"`
	Confidence      float64  `json:"confidence"`
}

type QAUpsertRequest struct {
	DocumentID      uint     `json:"documentId" binding:"required"`
	SourceSegmentID *uint    `json:"sourceSegmentId"`
	Question        string   `json:"question" binding:"required"`
	Answer          string   `json:"answer" binding:"required"`
	Tags            []string `json:"tags"`
	Enabled         bool     `json:"enabled"`
}

type StatusRequest struct {
	Enabled bool `json:"enabled"`
}

type BatchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required"`
}

type SegmentUpdateRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}

type GenerateAnswerRequest struct {
	DocumentID uint   `json:"documentId" binding:"required"`
	Question   string `json:"question" binding:"required"`
}

type KnowledgeBaseCreateRequest struct {
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

type KnowledgeBaseUpdateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type KnowledgeBaseIndexRequest struct {
	IndexType   string         `json:"indexType" binding:"required"`
	IndexParams map[string]any `json:"indexParams"`
}

type SearchRequest struct {
	KBID  uint   `json:"kbId" binding:"required"`
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"topK"`
}

type KnowledgeAskRequest struct {
	KnowledgeBaseID uint   `json:"knowledgeBaseId" binding:"required"`
	Question        string `json:"question" binding:"required"`
	RecallCount     int    `json:"recallCount"`
	RerankMode      string `json:"rerankMode"`
	UseCount        int    `json:"useCount"`
}

type DocumentUpdateRequest struct {
	OriginalName  string `json:"originalName"`
	ChunkStrategy string `json:"chunkStrategy"`
	ChunkSize     *int   `json:"chunkSize"`
	ChunkOverlap  *int   `json:"chunkOverlap"`
}

type DocumentUploadRequest struct {
	KnowledgeBaseID uint   `json:"knowledgeBaseId" binding:"required"`
	ChunkStrategy   string `json:"chunkStrategy"`
	ChunkSize       int    `json:"chunkSize"`
	ChunkOverlap    int    `json:"chunkOverlap"`
}
