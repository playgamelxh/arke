export type DocumentStatus = 'uploaded' | 'parsing' | 'parsed' | 'failed'

export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PageResponse<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface DocumentItem {
  id: number
  knowledgeBaseId?: number | null
  name: string
  originalName: string
  fileType: string
  fileSize: number
  status: DocumentStatus
  parseError: string
  chunkStrategy: ChunkStrategy
  chunkSize: number
  chunkOverlap: number
  segmentCount: number
  qaCount: number
  createdAt: string
  updatedAt: string
}

export interface DocumentSegment {
  id: number
  documentId: number
  vectorId?: string
  segmentType: 'document' | 'page' | 'slide' | 'sheet' | 'paragraph' | 'chunk'
  segmentIndex: number
  title: string
  content: string
  indexedAt?: string | null
  createdAt: string
}

export interface GeneratedQAItem {
  question: string
  answer: string
  keywords?: string[]
  documentId: number
  sourceSegmentId?: number
  sourceExcerpt: string
  confidence: number
}

export interface GenerateAnswerResult {
  answer: string
  sourceExcerpt: string
  sourceSegmentId?: number
  confidence: number
}

export type QAGenerateTaskStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface QAGenerateTask {
  id: number
  knowledgeBaseId: number
  documentId: number
  status: QAGenerateTaskStatus
  progress: number
  message: string
  targetCount: number
  generatedCount: number
  currentBatch: number
  totalBatches: number
  batchSize?: number
  items?: GeneratedQAItem[]
  error?: string
  createdAt: string
  updatedAt: string
}

export interface QAItem {
  id: number
  documentId: number
  documentName: string
  sourceSegmentId?: number
  question: string
  answer: string
  tags: string[]
  enabled: boolean
  confidence: number
  createdAt: string
  updatedAt: string
}

export interface Stats {
  documents: number
  parsed: number
  failed: number
  qa: number
  recentDocuments: DocumentItem[]
}

export interface SettingsMap {
  maxFileSizeMB: string
  allowedTypes: string
  defaultQACount: string
  qaGenerateBatchSize: string
  modelEndpoint: string
  modelName?: string
  storageMode: 'local' | 'rustfs'
  localUploadDir: string
  rustfsEndpoint: string
  rustfsAccessKey: string
  rustfsSecretKey: string
  rustfsBucket: string
  rustfsRegion: string
  rustfsUseSSL: string
  parseEngine: 'auto' | 'mineru' | 'native'
  parsePDFNativeFallback: string
  mineruBaseURL: string
  mineruTimeoutSeconds: string
  mineruParseMethod: 'auto' | 'txt' | 'ocr'
  mineruEffort: 'low' | 'medium' | 'high'
  mineruLanguage: string
  mineruImageAnalysis: string
  mineruTableEnable: string
  mineruFormulaEnable: string
}

export type ChunkStrategy = 'none' | 'paragraph' | 'fixed' | 'sentence'
export type IndexType = 'HNSW' | 'IVF_FLAT' | 'ANNOY' | 'FLAT'

export interface KnowledgeBase {
  id: number
  name: string
  description: string
  embeddingModel: string
  embeddingDim: number
  indexType: IndexType
  indexParams: string
  chunkStrategy: ChunkStrategy
  chunkSize: number
  chunkOverlap: number
  milvusCollection: string
  docCount: number
  vectorCount: number
  createdAt: string
  updatedAt: string
}

export interface SearchHit {
  id: string
  distance: number
  docId: string
  segmentId: string
  content: string
}

export interface KnowledgeAskSource {
  documentId: number
  sourceSegmentId?: number
  content: string
  score: number
  originalDistance: number
}

export interface KnowledgeAskResult {
  answer: string
  confidence: number
  sources: KnowledgeAskSource[]
  recallCount: number
  useCount: number
  rerankMode: string
  sourceExcerpt: string
}
