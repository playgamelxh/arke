package models

type DocumentStatus string

const (
	StatusUploaded DocumentStatus = "uploaded"
	StatusParsing  DocumentStatus = "parsing"
	StatusParsed   DocumentStatus = "parsed"
	StatusFailed   DocumentStatus = "failed"
)

type QAGenerateTaskStatus string

const (
	QATaskPending   QAGenerateTaskStatus = "pending"
	QATaskRunning   QAGenerateTaskStatus = "running"
	QATaskCompleted QAGenerateTaskStatus = "completed"
	QATaskFailed    QAGenerateTaskStatus = "failed"
)

type ChunkStrategy string

const (
	ChunkStrategyNone      ChunkStrategy = "none"
	ChunkStrategyParagraph ChunkStrategy = "paragraph"
	ChunkStrategyFixed     ChunkStrategy = "fixed"
	ChunkStrategySentence  ChunkStrategy = "sentence"
)

func (c ChunkStrategy) Valid() bool {
	switch c {
	case ChunkStrategyNone, ChunkStrategyParagraph, ChunkStrategyFixed, ChunkStrategySentence:
		return true
	}
	return false
}

type IndexType string

const (
	IndexHNSW    IndexType = "HNSW"
	IndexIVFFlat IndexType = "IVF_FLAT"
	IndexANNOY   IndexType = "ANNOY"
	IndexFlat    IndexType = "FLAT"
)

func (i IndexType) Valid() bool {
	switch i {
	case IndexHNSW, IndexIVFFlat, IndexANNOY, IndexFlat:
		return true
	}
	return false
}

func (i IndexType) DefaultParams() map[string]any {
	switch i {
	case IndexHNSW:
		return map[string]any{"M": 16, "efConstruction": 200}
	case IndexIVFFlat:
		return map[string]any{"nlist": 128}
	case IndexANNOY:
		return map[string]any{"n_trees": 8}
	default:
		return map[string]any{}
	}
}
