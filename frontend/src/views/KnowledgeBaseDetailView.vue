<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  Database,
  FileText,
  Loader2,
  Search,
  Trash2,
  Upload,
  X,
} from 'lucide-vue-next'
import { api } from '@/api/client'
import type { DocumentItem, KnowledgeBase, SearchHit } from '@/types/domain'
import { confirm } from '@/composables/useConfirm'
import { toast } from '@/composables/useToast'

const route = useRoute()
const router = useRouter()

const kbId = computed(() => Number(route.params.id))
const kb = ref<KnowledgeBase | null>(null)
const documents = ref<DocumentItem[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showUpload = ref(false)
const uploadFile = ref<File | null>(null)
const uploadStrategy = ref<'paragraph' | 'fixed' | 'sentence' | 'none'>('paragraph')
const uploadChunkSize = ref(500)
const uploadChunkOverlap = ref(50)
const uploading = ref(false)

const showSearch = ref(false)
const searchQuery = ref('')
const searchTopK = ref(5)
const searchResults = ref<SearchHit[]>([])
const searching = ref(false)

const parsingDocs = ref<Set<number>>(new Set())
const indexingDocs = ref<Set<number>>(new Set())

const showChunkSettings = ref(false)
const editingDoc = ref<DocumentItem | null>(null)
const editChunkStrategy = ref<'paragraph' | 'fixed' | 'sentence' | 'none'>('paragraph')
const editChunkSize = ref(500)
const editChunkOverlap = ref(50)
const savingChunkSettings = ref(false)

const loadKB = async () => {
  try {
    kb.value = await api.knowledgeBase(kbId.value)
  } catch (e) {
    console.error(e)
  }
}

const loadDocuments = async () => {
  loading.value = true
  try {
    const res = await api.knowledgeBaseDocuments(kbId.value, { page: page.value, pageSize: pageSize.value })
    documents.value = res.list
    total.value = res.total
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const goToDocument = (doc: DocumentItem) => {
  router.push(`/documents/${doc.id}?kbId=${kbId.value}`)
}

const openUpload = () => {
  uploadFile.value = null
  uploadStrategy.value = (kb.value?.chunkStrategy as any) || 'paragraph'
  uploadChunkSize.value = kb.value?.chunkSize || 500
  uploadChunkOverlap.value = kb.value?.chunkOverlap || 50
  showUpload.value = true
}

const onFileChange = (e: Event) => {
  const target = e.target as HTMLInputElement
  uploadFile.value = target.files?.[0] || null
}

const submitUpload = async () => {
  if (!uploadFile.value) {
    toast.warning('请选择文件')
    return
  }
  uploading.value = true
  try {
    await api.uploadDocumentToKB(kbId.value, uploadFile.value, {
      chunkStrategy: uploadStrategy.value,
      chunkSize: uploadChunkSize.value,
      chunkOverlap: uploadChunkOverlap.value,
    })
    showUpload.value = false
    toast.success('文档上传成功')
    await loadKB()
    await loadDocuments()
  } catch (e: any) {
    toast.error(e.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

const openChunkSettings = (doc: DocumentItem) => {
  editingDoc.value = doc
  editChunkStrategy.value = (doc.chunkStrategy || kb.value?.chunkStrategy || 'paragraph') as 'paragraph' | 'fixed' | 'sentence' | 'none'
  editChunkSize.value = doc.chunkSize || kb.value?.chunkSize || 500
  editChunkOverlap.value = doc.chunkOverlap ?? kb.value?.chunkOverlap ?? 50
  showChunkSettings.value = true
}

const saveChunkSettings = async () => {
  if (!editingDoc.value) return
  if (editChunkStrategy.value === 'fixed') {
    if (editChunkSize.value < 100 || editChunkSize.value > 5000) {
      toast.warning('切片大小需在 100-5000 之间')
      return
    }
    if (editChunkOverlap.value < 0 || editChunkOverlap.value >= editChunkSize.value) {
      toast.warning('切片重叠必须大于等于 0 且小于切片大小')
      return
    }
  }
  savingChunkSettings.value = true
  try {
    await api.updateDocument(editingDoc.value.id, {
      chunkStrategy: editChunkStrategy.value,
      chunkSize: editChunkSize.value,
      chunkOverlap: editChunkOverlap.value,
    })
    showChunkSettings.value = false
    toast.success('文档切片策略已保存，正在按新策略重新解析并更新向量库')
    await parseDocument(editingDoc.value)
  } catch (e: any) {
    toast.error(e.message || '保存失败')
  } finally {
    savingChunkSettings.value = false
  }
}

const parseDocument = async (doc: DocumentItem) => {
  if (parsingDocs.value.has(doc.id)) return
  parsingDocs.value.add(doc.id)
  try {
    await api.reparseDocument(doc.id)
    toast.success('文档解析已启动')
    await pollDocumentStatus(doc.id)
    await loadKB()
    await loadDocuments()
  } catch (e: any) {
    toast.error(e.message || '解析失败')
  } finally {
    parsingDocs.value.delete(doc.id)
  }
}

const indexDocument = async (doc: DocumentItem) => {
  if (indexingDocs.value.has(doc.id)) return
  indexingDocs.value.add(doc.id)
  try {
    await api.indexDocument(doc.id)
    toast.success('文档索引已启动')
    await pollDocumentStatus(doc.id)
    await loadKB()
  } catch (e: any) {
    toast.error(e.message || '索引失败')
  } finally {
    indexingDocs.value.delete(doc.id)
  }
}

const pollDocumentStatus = async (docId: number) => {
  const maxAttempts = 60
  const interval = 3000
  for (let i = 0; i < maxAttempts; i++) {
    await new Promise(resolve => setTimeout(resolve, interval))
    try {
      const doc = await api.document(docId)
      if (doc.status === 'parsed' || doc.status === 'failed') {
        return
      }
    } catch {
      // ignore errors during polling
    }
  }
}

const deleteDocument = async (doc: DocumentItem) => {
  if (!await confirm(`确定要删除文档「${doc.originalName}」吗？相关向量数据也会被删除。`)) return
  try {
    await api.deleteDocument(doc.id)
    toast.success('文档已删除')
    await loadKB()
    await loadDocuments()
  } catch (e: any) {
    toast.error(e.message || '删除失败')
  }
}

const openSearch = () => {
  searchQuery.value = ''
  searchResults.value = []
  showSearch.value = true
}

const submitSearch = async () => {
  if (!searchQuery.value.trim()) return
  searching.value = true
  try {
    const result = await api.searchKB(kbId.value, searchQuery.value, searchTopK.value)
    searchResults.value = [...result].sort((a, b) => a.distance - b.distance)
  } catch (e: any) {
    toast.error(e.message || '搜索失败')
  } finally {
    searching.value = false
  }
}

const statusLabel = (s: string) => {
  return {
    uploaded: '已上传',
    parsing: '解析中',
    parsed: '已解析',
    failed: '解析失败',
  }[s] || s
}

const statusColor = (s: string) => {
  return {
    uploaded: 'bg-indigo-400/15 text-indigo-200 border-indigo-400/20',
    parsing: 'bg-amber-400/15 text-amber-200 border-amber-400/20',
    parsed: 'bg-emerald-400/15 text-emerald-200 border-emerald-400/20',
    failed: 'bg-rose-400/15 text-rose-200 border-rose-400/20',
  }[s] || 'bg-slate-400/15 text-slate-200 border-slate-400/20'
}

const formatFileSize = (size: number) => {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(1) + ' KB'
  return (size / 1024 / 1024).toFixed(1) + ' MB'
}

const chunkStrategyLabel = (s: string) => {
  const map: Record<string, string> = {
    paragraph: '段落切片',
    fixed: '定长切片',
    sentence: '句子切片',
    none: '整体不切片',
  }
  return map[s] || s
}

onMounted(async () => {
  await loadKB()
  await loadDocuments()
})
</script>

<template>
  <div class="kb-detail-view">
    <div class="mb-6 flex items-center gap-4">
      <button
        class="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-slate-400 transition hover:bg-white/5 hover:text-slate-200"
        @click="router.push('/knowledge-bases')"
      >
        <ArrowLeft class="h-4 w-4" />
        返回
      </button>
    </div>

    <div v-if="kb" class="mb-8 flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
      <div class="flex-1">
        <div class="flex items-center gap-4">
          <div class="flex h-14 w-14 items-center justify-center rounded-2xl bg-cyan-400/15 text-cyan-300">
            <Database class="h-7 w-7" />
          </div>
          <div>
            <h1 class="text-2xl font-semibold text-slate-100">{{ kb.name }}</h1>
            <p class="mt-1 text-sm text-slate-400">{{ kb.description || '暂无描述' }}</p>
          </div>
        </div>
      </div>
      <div class="flex gap-3">
        <button
          class="flex items-center gap-2 rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-medium text-slate-200 transition hover:bg-white/10"
          @click="openSearch"
        >
          <Search class="h-4 w-4" />
          检索测试
        </button>
        <button
          class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300"
          @click="openUpload"
        >
          <Upload class="h-4 w-4" />
          上传文档
        </button>
      </div>
    </div>

    <div v-if="kb" class="mb-8 grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-5">
      <div class="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
        <p class="text-xs text-slate-500">文档数</p>
        <p class="mt-2 text-2xl font-semibold text-slate-100">{{ kb.docCount }}</p>
      </div>
      <div class="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
        <p class="text-xs text-slate-500">向量数</p>
        <p class="mt-2 text-2xl font-semibold text-slate-100">{{ kb.vectorCount }}</p>
      </div>
      <div class="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
        <p class="text-xs text-slate-500">索引类型</p>
        <p class="mt-2 text-lg font-semibold text-slate-100">{{ kb.indexType }}</p>
      </div>
      <div class="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
        <p class="text-xs text-slate-500">切片策略</p>
        <p class="mt-2 text-lg font-semibold text-slate-100">{{ chunkStrategyLabel(kb.chunkStrategy) }}</p>
      </div>
      <div class="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
        <p class="text-xs text-slate-500">Embedding</p>
        <p class="mt-2 text-sm font-semibold text-slate-100">{{ kb.embeddingModel }}</p>
        <p class="text-xs text-slate-500">{{ kb.embeddingDim }} 维</p>
      </div>
    </div>

    <div class="mb-4 flex items-center justify-between">
      <h2 class="text-lg font-semibold text-slate-100">文档列表</h2>
      <span class="text-sm text-slate-400">共 {{ total }} 个文档</span>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 animate-spin text-cyan-400" />
      <span class="ml-3 text-slate-400">加载中...</span>
    </div>

    <div v-else-if="documents.length === 0" class="flex flex-col items-center justify-center py-20">
      <div class="flex h-20 w-20 items-center justify-center rounded-3xl bg-white/[0.03]">
        <FileText class="h-10 w-10 text-slate-500" />
      </div>
      <p class="mt-6 text-lg font-medium text-slate-200">该知识库下还没有文档</p>
      <button
        class="mt-6 flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300"
        @click="openUpload"
      >
        <Upload class="h-4 w-4" />
        上传文档
      </button>
    </div>

    <div v-else class="overflow-hidden rounded-3xl border border-white/10 bg-white/[0.02]">
      <table class="w-full text-sm">
        <thead class="border-b border-white/5 bg-white/[0.02]">
          <tr class="text-left text-xs font-medium text-slate-400">
            <th class="px-5 py-4">名称</th>
            <th class="px-5 py-4">类型</th>
            <th class="px-5 py-4">大小</th>
            <th class="px-5 py-4">切片策略</th>
            <th class="px-5 py-4">切片数</th>
            <th class="px-5 py-4">状态</th>
            <th class="px-5 py-4">创建时间</th>
            <th class="px-5 py-4 text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="doc in documents"
            :key="doc.id"
            class="border-b border-white/5 transition last:border-b-0 hover:bg-white/[0.02]"
          >
            <td class="px-5 py-4">
              <a
                class="cursor-pointer font-medium text-slate-100 transition hover:text-cyan-300"
                @click.prevent="goToDocument(doc)"
              >
                {{ doc.originalName }}
              </a>
            </td>
            <td class="px-5 py-4 text-slate-300">{{ doc.fileType.toUpperCase() }}</td>
            <td class="px-5 py-4 text-slate-300">{{ formatFileSize(doc.fileSize) }}</td>
            <td class="px-5 py-4 text-slate-300">
              <div class="flex flex-col gap-1">
                <span>{{ chunkStrategyLabel(doc.chunkStrategy || kb?.chunkStrategy || 'paragraph') }}</span>
                <span class="text-xs text-slate-500">{{ doc.chunkSize || kb?.chunkSize || 500 }} / {{ doc.chunkOverlap ?? kb?.chunkOverlap ?? 50 }}</span>
              </div>
            </td>
            <td class="px-5 py-4 text-slate-300">{{ doc.segmentCount }}</td>
            <td class="px-5 py-4">
              <span
                class="inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-medium"
                :class="statusColor(doc.status)"
              >
                <span
                  class="mr-1.5 h-1.5 w-1.5 rounded-full"
                  :class="doc.status === 'parsing' ? 'animate-pulse' : ''"
                  :style="{ background: doc.status === 'uploaded' ? '#a5b4fc' : doc.status === 'parsing' ? '#fcd34d' : doc.status === 'parsed' ? '#6ee7b7' : '#fda4af' }"
                ></span>
                {{ statusLabel(doc.status) }}
              </span>
            </td>
            <td class="px-5 py-4 text-slate-400">{{ new Date(doc.createdAt).toLocaleString() }}</td>
            <td class="px-5 py-4">
              <div class="flex items-center justify-end gap-2">
                <button
                  v-if="doc.status === 'uploaded' || doc.status === 'failed'"
                  class="flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-medium text-cyan-300 transition hover:bg-cyan-400/10 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="parsingDocs.has(doc.id)"
                  @click="parseDocument(doc)"
                >
                  <Loader2 v-if="parsingDocs.has(doc.id)" class="h-3 w-3 animate-spin" />
                  {{ parsingDocs.has(doc.id) ? '解析中...' : '解析' }}
                </button>
                <button
                  v-if="doc.status === 'parsed'"
                  class="flex items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-medium text-emerald-300 transition hover:bg-emerald-400/10 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="indexingDocs.has(doc.id)"
                  @click="indexDocument(doc)"
                >
                  <Loader2 v-if="indexingDocs.has(doc.id)" class="h-3 w-3 animate-spin" />
                  {{ indexingDocs.has(doc.id) ? '索引中...' : '索引' }}
                </button>
                <button
                  class="rounded-xl px-3 py-1.5 text-xs font-medium text-cyan-200 transition hover:bg-cyan-400/10"
                  @click="openChunkSettings(doc)"
                >
                  切片设置
                </button>
                <button
                  class="rounded-xl px-3 py-1.5 text-xs font-medium text-slate-300 transition hover:bg-white/10"
                  @click="goToDocument(doc)"
                >
                  查看
                </button>
                <button
                  class="rounded-xl px-3 py-1.5 text-xs font-medium text-rose-300 transition hover:bg-rose-500/10"
                  @click="deleteDocument(doc)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="total > pageSize" class="mt-6 flex items-center justify-center gap-4">
      <button
        class="rounded-xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-slate-300 transition hover:bg-white/10 disabled:opacity-50"
        :disabled="page <= 1"
        @click="page--; loadDocuments()"
      >
        上一页
      </button>
      <span class="text-sm text-slate-400">第 {{ page }} 页 / 共 {{ Math.ceil(total / pageSize) }} 页</span>
      <button
        class="rounded-xl border border-white/10 bg-white/5 px-4 py-2 text-sm text-slate-300 transition hover:bg-white/10 disabled:opacity-50"
        :disabled="page * pageSize >= total"
        @click="page++; loadDocuments()"
      >
        下一页
      </button>
    </div>

    <Teleport to="body">
      <div
        v-if="showChunkSettings"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
        @click.self="showChunkSettings = false"
      >
        <div class="max-h-[90vh] w-full max-w-lg overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <div>
              <h3 class="text-xl font-semibold text-slate-100">文档切片设置</h3>
              <p class="mt-1 text-sm text-slate-500">{{ editingDoc?.originalName }}</p>
            </div>
            <button
              class="rounded-xl bg-white/10 p-2 text-slate-300 transition hover:bg-white/15 hover:text-white"
              @click="showChunkSettings = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="space-y-5">
            <div class="rounded-2xl border border-cyan-300/15 bg-cyan-300/10 p-4 text-sm leading-6 text-cyan-50">
              默认继承知识库切片策略；这里保存后，该文档会使用自己的切片策略，并立即重新解析、重新写入向量库。
            </div>
            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">切片策略</label>
              <select
                v-model="editChunkStrategy"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              >
                <option value="paragraph">段落切片</option>
                <option value="fixed">定长切片</option>
                <option value="sentence">句子切片</option>
                <option value="none">整体不切片</option>
              </select>
            </div>
            <template v-if="editChunkStrategy === 'fixed'">
              <div>
                <label class="mb-2 block text-sm font-medium text-slate-200">切片大小（字符数）</label>
                <input
                  v-model.number="editChunkSize"
                  type="number"
                  min="100"
                  max="5000"
                  class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-slate-200">切片重叠（字符数）</label>
                <input
                  v-model.number="editChunkOverlap"
                  type="number"
                  min="0"
                  :max="editChunkSize - 1"
                  class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
            </template>
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button
              class="rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-medium text-slate-300 transition hover:bg-white/10"
              @click="showChunkSettings = false"
            >
              取消
            </button>
            <button
              class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300 disabled:opacity-60"
              :disabled="savingChunkSettings"
              @click="saveChunkSettings"
            >
              <Loader2 v-if="savingChunkSettings" class="h-4 w-4 animate-spin" />
              {{ savingChunkSettings ? '保存中...' : '保存并重解析' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="showUpload"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
        @click.self="showUpload = false"
      >
        <div class="max-h-[90vh] w-full max-w-lg overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-xl font-semibold text-slate-100">上传文档</h3>
            <button
              class="rounded-xl bg-white/10 p-2 text-slate-300 transition hover:bg-white/15 hover:text-white"
              @click="showUpload = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="space-y-5">
            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">选择文件（最大 500MB）</label>
              <div class="relative">
                <input
                  type="file"
                  accept=".pdf,.docx,.doc,.md,.txt,.xlsx,.xls"
                  class="w-full cursor-pointer rounded-2xl border border-white/10 bg-slate-950 px-4 py-8 text-sm text-slate-400 file:mr-4 file:rounded-xl file:border-0 file:bg-cyan-400/15 file:px-4 file:py-2 file:text-sm file:font-medium file:text-cyan-200 hover:file:bg-cyan-400/25"
                  @change="onFileChange"
                />
              </div>
              <p v-if="uploadFile" class="mt-2 text-sm text-cyan-300">
                已选择：{{ uploadFile.name }}
              </p>
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">切片策略（默认使用知识库配置，可按本次上传单独调整）</label>
              <select
                v-model="uploadStrategy"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              >
                <option value="paragraph">段落切片</option>
                <option value="fixed">定长切片</option>
                <option value="sentence">句子切片</option>
                <option value="none">整体不切片</option>
              </select>
            </div>

            <template v-if="uploadStrategy === 'fixed'">
              <div>
                <label class="mb-2 block text-sm font-medium text-slate-200">切片大小（字符数）</label>
                <input
                  v-model.number="uploadChunkSize"
                  type="number"
                  min="100"
                  max="5000"
                  class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-slate-200">切片重叠（字符数）</label>
                <input
                  v-model.number="uploadChunkOverlap"
                  type="number"
                  min="0"
                  :max="uploadChunkSize - 1"
                  class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
            </template>
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button
              class="rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-medium text-slate-300 transition hover:bg-white/10"
              @click="showUpload = false"
            >
              取消
            </button>
            <button
              class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300 disabled:opacity-60"
              :disabled="uploading"
              @click="submitUpload"
            >
              <Loader2 v-if="uploading" class="h-4 w-4 animate-spin" />
              {{ uploading ? '上传中...' : '上传' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="showSearch"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
        @click.self="showSearch = false"
      >
        <div class="max-h-[90vh] w-full max-w-3xl overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-xl font-semibold text-slate-100">检索测试</h3>
            <button
              class="rounded-xl bg-white/10 p-2 text-slate-300 transition hover:bg-white/15 hover:text-white"
              @click="showSearch = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="mb-5 flex gap-3">
            <input
              v-model="searchQuery"
              placeholder="输入查询内容"
              class="flex-1 rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              @keyup.enter="submitSearch"
            />
            <input
              v-model.number="searchTopK"
              type="number"
              min="1"
              max="20"
              class="w-20 rounded-2xl border border-white/10 bg-slate-950 px-3 py-3 text-sm text-center text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
            />
            <button
              class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300 disabled:opacity-60"
              :disabled="searching"
              @click="submitSearch"
            >
              <Loader2 v-if="searching" class="h-4 w-4 animate-spin" />
              {{ searching ? '搜索中' : '搜索' }}
            </button>
          </div>

          <div v-if="searchResults.length === 0 && !searching" class="flex flex-col items-center justify-center py-16">
            <Search class="h-12 w-12 text-slate-600" />
            <p class="mt-4 text-sm text-slate-500">输入查询后点击搜索</p>
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="(hit, idx) in searchResults"
              :key="idx"
              class="rounded-2xl border border-white/10 bg-white/[0.03] p-4"
            >
              <div class="mb-3 flex items-center gap-3 text-xs">
                <span class="rounded-xl bg-cyan-400/15 px-2.5 py-1 font-medium text-cyan-200">
                  #{{ idx + 1 }}
                </span>
                <span class="rounded-xl bg-emerald-400/15 px-2.5 py-1 font-medium text-emerald-200">
                  相似度：{{ (1 - hit.distance).toFixed(4) }}
                </span>
                <span class="text-slate-500">
                  doc_id={{ hit.docId }} / segment_id={{ hit.segmentId }}
                </span>
              </div>
              <div class="text-sm leading-7 text-slate-200">{{ hit.content }}</div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
</style>
