import type {
  ApiResponse,
  DocumentItem,
  DocumentSegment,
  GenerateAnswerResult,
  GeneratedQAItem,
  KnowledgeBase,
  KnowledgeAskResult,
  PageResponse,
  QAItem,
  QAGenerateTask,
  SearchHit,
  SettingsMap,
  Stats,
} from '@/types/domain'

const API_BASE = import.meta.env.VITE_API_BASE || '/api'

interface ListParams {
  page?: number
  pageSize?: number
  keyword?: string
  status?: string
  documentId?: number | string
  enabled?: string
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: options.body instanceof FormData ? undefined : { 'Content-Type': 'application/json' },
    ...options,
  })
  const text = await response.text()
  if (!text) {
    if (response.status === 413) {
      throw new Error('文件过大，超过上传大小限制，请在系统设置中调整或压缩文件后重试')
    }
    if (response.status === 502 || response.status === 504) {
      throw new Error('请求超时，问答生成耗时较长，请稍后重试')
    }
    throw new Error(`服务器返回空响应（HTTP ${response.status}）`)
  }
  let payload: ApiResponse<T>
  try {
    payload = JSON.parse(text) as ApiResponse<T>
  } catch {
    throw new Error(`服务器响应格式异常（HTTP ${response.status}）`)
  }
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || '请求失败')
  }
  return payload.data
}

function query(params: ListParams) {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') {
      search.set(key, String(value))
    }
  })
  const text = search.toString()
  return text ? `?${text}` : ''
}

export const api = {
  stats: () => request<Stats>('/stats'),
  uploadDocument: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return request<DocumentItem>('/documents/upload', { method: 'POST', body: form })
  },
  documents: (params: ListParams = {}) => request<PageResponse<DocumentItem>>(`/documents${query(params)}`),
  document: (id: number) => request<DocumentItem>(`/documents/${id}`),
  segments: (id: number) => request<DocumentSegment[]>(`/documents/${id}/segments`),
  updateSegment: (documentId: number, segmentId: number, payload: { title?: string; content: string }) =>
    request<DocumentSegment>(`/documents/${documentId}/segments/${segmentId}`, { method: 'PUT', body: JSON.stringify(payload) }),
  updateDocument: (id: number, payload: { originalName?: string; chunkStrategy?: string; chunkSize?: number; chunkOverlap?: number }) =>
    request<DocumentItem>(`/documents/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  reparseDocument: (id: number) => request<DocumentItem>(`/documents/${id}/parse`, { method: 'POST' }),
  deleteDocument: (id: number) => request<{ deleted: boolean }>(`/documents/${id}`, { method: 'DELETE' }),
  generatePreview: (payload: { knowledgeBaseId: number; count: number; difficulty: string; instruction?: string; overwrite: boolean }) =>
    request<QAGenerateTask>('/qa/generate-preview', { method: 'POST', body: JSON.stringify(payload) }),
  generateTask: (id: number) => request<QAGenerateTask>(`/qa/generate-tasks/${id}`),
  saveGenerated: (payload: { knowledgeBaseId: number; items: GeneratedQAItem[]; overwrite: boolean }) =>
    request<{ saved: number }>('/qa/save-generated', { method: 'POST', body: JSON.stringify(payload) }),
  generateAnswer: (payload: { documentId: number; question: string }) =>
    request<GenerateAnswerResult>('/qa/generate-answer', { method: 'POST', body: JSON.stringify(payload) }),
  qaList: (params: ListParams = {}) => request<PageResponse<QAItem>>(`/qa${query(params)}`),
  createQA: (payload: Partial<QAItem> & { documentId: number; question: string; answer: string }) =>
    request<QAItem>('/qa', { method: 'POST', body: JSON.stringify(payload) }),
  updateQA: (id: number, payload: Partial<QAItem> & { documentId: number; question: string; answer: string }) =>
    request<QAItem>(`/qa/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  updateQAStatus: (id: number, enabled: boolean) =>
    request<QAItem>(`/qa/${id}/status`, { method: 'PATCH', body: JSON.stringify({ enabled }) }),
  deleteQA: (id: number) => request<{ deleted: boolean }>(`/qa/${id}`, { method: 'DELETE' }),
  batchDeleteQA: (ids: number[]) => request<{ deleted: number }>('/qa/batch', { method: 'DELETE', body: JSON.stringify({ ids }) }),
  settings: () => request<SettingsMap>('/settings'),
  saveSettings: (payload: SettingsMap) => request<SettingsMap>('/settings', { method: 'PUT', body: JSON.stringify(payload) }),
  testRustFS: (payload: Partial<Pick<SettingsMap, 'rustfsEndpoint' | 'rustfsAccessKey' | 'rustfsSecretKey' | 'rustfsBucket' | 'rustfsRegion' | 'rustfsUseSSL'>>) =>
    request<{ ok: boolean; message: string }>('/settings/test-rustfs', { method: 'POST', body: JSON.stringify(payload) }),
  testParseSettings: (payload: Partial<Pick<SettingsMap, 'parseEngine' | 'parsePDFNativeFallback' | 'mineruBaseURL' | 'mineruTimeoutSeconds' | 'mineruParseMethod' | 'mineruEffort' | 'mineruLanguage' | 'mineruImageAnalysis' | 'mineruTableEnable' | 'mineruFormulaEnable'>>) =>
    request<{ ok: boolean; message: string }>('/settings/test-parse', { method: 'POST', body: JSON.stringify(payload) }),

  // 知识库管理
  knowledgeBases: () => request<KnowledgeBase[]>('/knowledge-bases'),
  embeddingModels: () => request<{ model: string; dimensions: number[] }[]>('/knowledge-bases/embedding-models'),
  knowledgeBase: (id: number) => request<KnowledgeBase>(`/knowledge-bases/${id}`),
  createKnowledgeBase: (payload: Partial<KnowledgeBase> & { name: string }) =>
    request<KnowledgeBase>('/knowledge-bases', { method: 'POST', body: JSON.stringify(payload) }),
  updateKnowledgeBase: (id: number, payload: { name?: string; description?: string }) =>
    request<KnowledgeBase>(`/knowledge-bases/${id}`, { method: 'PUT', body: JSON.stringify(payload) }),
  updateKBIndex: (id: number, payload: { indexType: string; indexParams?: any }) =>
    request<{ status: string }>(`/knowledge-bases/${id}/index`, { method: 'PUT', body: JSON.stringify(payload) }),
  deleteKnowledgeBase: (id: number) => request<{ status: string }>(`/knowledge-bases/${id}`, { method: 'DELETE' }),
  knowledgeBaseDocuments: (id: number, params: ListParams = {}) =>
    request<PageResponse<DocumentItem>>(`/knowledge-bases/${id}/documents${query(params)}`),
  searchKB: (kbId: number, query: string, topK = 5) =>
    request<SearchHit[]>('/knowledge-bases/search', { method: 'POST', body: JSON.stringify({ kbId, query, topK }) }),
  knowledgeAsk: (payload: { knowledgeBaseId: number; question: string; recallCount: number; rerankMode: string; useCount: number }) =>
    request<KnowledgeAskResult>('/knowledge-ask', { method: 'POST', body: JSON.stringify(payload) }),

  // 文档操作
  uploadDocumentToKB: (kbId: number, file: File, options: { chunkStrategy?: string; chunkSize?: number; chunkOverlap?: number } = {}) => {
    const form = new FormData()
    form.append('file', file)
    form.append('knowledgeBaseId', String(kbId))
    if (options.chunkStrategy) form.append('chunkStrategy', options.chunkStrategy)
    if (options.chunkSize) form.append('chunkSize', String(options.chunkSize))
    if (options.chunkOverlap) form.append('chunkOverlap', String(options.chunkOverlap))
    return request<DocumentItem>('/documents/upload', { method: 'POST', body: form })
  },
  indexDocument: (id: number) => request<{ status: string }>(`/documents/${id}/index`, { method: 'POST' }),
}
