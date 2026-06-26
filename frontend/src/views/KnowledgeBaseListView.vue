<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Database, Edit3, Settings, Trash2, X, Plus, Loader2 } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { ChunkStrategy, IndexType, KnowledgeBase } from '@/types/domain'
import { confirm } from '@/composables/useConfirm'
import { toast } from '@/composables/useToast'

const router = useRouter()

const knowledgeBases = ref<KnowledgeBase[]>([])
const loading = ref(false)
const showCreate = ref(false)
const showIndexEdit = ref(false)
const editingIndexKB = ref<KnowledgeBase | null>(null)
const showRename = ref(false)
const renamingKB = ref<KnowledgeBase | null>(null)
const renameInput = ref('')
const creating = ref(false)
const savingIndex = ref(false)
const savingRename = ref(false)

const form = reactive({
  name: '',
  description: '',
  embeddingModel: 'text-embedding-v3',
  embeddingDim: 1024,
  chunkStrategy: 'paragraph' as ChunkStrategy,
  chunkSize: 500,
  chunkOverlap: 50,
  indexType: 'HNSW' as IndexType,
})

const embeddingModelOptions = ref<{ model: string; dimensions: number[] }[]>([
  { model: 'text-embedding-v3', dimensions: [1024, 768, 512, 256, 128, 64] },
  { model: 'text-embedding-v4', dimensions: [2048, 1536, 1024, 768, 512, 256, 128, 64] },
])

const embeddingModelLabels: Record<string, string> = {
  'text-embedding-v3': 'text-embedding-v3（通用，1024 维）',
  'text-embedding-v4': 'text-embedding-v4（新一代，支持更高维度）',
}

const currentDims = computed(() => {
  return embeddingModelOptions.value.find(o => o.model === form.embeddingModel)?.dimensions || [1024]
})

const onEmbeddingModelChange = () => {
  if (!currentDims.value.includes(form.embeddingDim)) {
    form.embeddingDim = currentDims.value.includes(1024) ? 1024 : currentDims.value[0]
  }
}

const indexForm = reactive({
  indexType: 'HNSW' as IndexType,
})

const indexTypeOptions: { value: IndexType; label: string; description: string }[] = [
  { value: 'HNSW', label: 'HNSW', description: '图索引，速度快、精度高（推荐）' },
  { value: 'IVF_FLAT', label: 'IVF_FLAT', description: '倒排索引，适合超大数据集' },
  { value: 'ANNOY', label: 'ANNOY', description: '树形索引，内存占用低' },
  { value: 'FLAT', label: 'FLAT', description: '暴力搜索，精度最高' },
]

const chunkStrategyOptions: { value: ChunkStrategy; label: string; description: string }[] = [
  { value: 'paragraph', label: '段落切片（推荐）', description: '按段落切分，长段落自动细分' },
  { value: 'fixed', label: '定长切片', description: '按固定字符数切分' },
  { value: 'sentence', label: '句子切片', description: '按句子切分' },
  { value: 'none', label: '整体不切片', description: '将整个文档作为一个切片' },
]

const indexTypeLabel = (t: IndexType) => indexTypeOptions.find(o => o.value === t)?.label || t
const chunkStrategyLabel = (s: ChunkStrategy) => chunkStrategyOptions.find(o => o.value === s)?.label || s

const loadList = async () => {
  loading.value = true
  try {
    knowledgeBases.value = await api.knowledgeBases()
  } catch (e) {
    console.error(e)
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  form.name = ''
  form.description = ''
  form.embeddingModel = 'text-embedding-v3'
  form.embeddingDim = 1024
  form.chunkStrategy = 'paragraph'
  form.chunkSize = 500
  form.chunkOverlap = 50
  form.indexType = 'HNSW'
  showCreate.value = true
}

const submitCreate = async () => {
  if (!form.name.trim()) {
    toast.warning('请输入知识库名称')
    return
  }
  creating.value = true
  try {
    const kb = await api.createKnowledgeBase({
      name: form.name,
      description: form.description,
      embeddingModel: form.embeddingModel,
      embeddingDim: form.embeddingDim,
      chunkStrategy: form.chunkStrategy,
      chunkSize: form.chunkSize,
      chunkOverlap: form.chunkOverlap,
      indexType: form.indexType,
    })
    showCreate.value = false
    toast.success('知识库创建成功')
    await loadList()
    router.push(`/knowledge-bases/${kb.id}`)
  } catch (e: any) {
    toast.error(e.message || '创建失败')
  } finally {
    creating.value = false
  }
}

const openKB = (kb: KnowledgeBase) => {
  router.push(`/knowledge-bases/${kb.id}`)
}

const openRename = (kb: KnowledgeBase) => {
  renamingKB.value = kb
  renameInput.value = kb.name
  showRename.value = true
}

const submitRename = async () => {
  if (!renamingKB.value || !renameInput.value.trim()) return
  if (renameInput.value === renamingKB.value.name) {
    showRename.value = false
    return
  }
  savingRename.value = true
  try {
    await api.updateKnowledgeBase(renamingKB.value.id, { name: renameInput.value.trim() })
    toast.success('知识库名称已更新')
    showRename.value = false
    await loadList()
  } catch (e: any) {
    toast.error(e.message || '修改失败')
  } finally {
    savingRename.value = false
  }
}

const deleteKB = async (kb: KnowledgeBase) => {
  if (!await confirm(`确定要删除知识库「${kb.name}」吗？该操作将删除所有相关文档和向量数据。`)) return
  try {
    await api.deleteKnowledgeBase(kb.id)
    toast.success('知识库已删除')
    await loadList()
  } catch (e: any) {
    toast.error(e.message || '删除失败')
  }
}

const openIndexEdit = (kb: KnowledgeBase) => {
  if (kb.docCount > 0 || kb.vectorCount > 0) {
    toast.warning('知识库已有数据，无法更换索引类型。请先清空知识库中的文档。')
    return
  }
  editingIndexKB.value = kb
  indexForm.indexType = kb.indexType
  showIndexEdit.value = true
}

const submitIndexEdit = async () => {
  if (!editingIndexKB.value) return
  savingIndex.value = true
  try {
    await api.updateKBIndex(editingIndexKB.value.id, { indexType: indexForm.indexType })
    showIndexEdit.value = false
    toast.success('索引类型已更新')
    await loadList()
  } catch (e: any) {
    toast.error(e.message || '更新失败')
  } finally {
    savingIndex.value = false
  }
}

const loadEmbeddingModels = async () => {
  try {
    const models = await api.embeddingModels()
    if (Array.isArray(models) && models.length > 0) {
      embeddingModelOptions.value = models
    }
  } catch (e) {
    console.error(e)
  }
}

onMounted(() => {
  loadList()
  loadEmbeddingModels()
})
</script>

<template>
  <div class="kb-list-view">
    <div class="mb-8 flex items-end justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-slate-100">知识库管理</h1>
        <p class="mt-2 text-sm text-slate-400">管理你的知识库，向量化文档以支持智能检索</p>
      </div>
      <button
        class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300"
        @click="openCreate"
      >
        <Plus class="h-4 w-4" />
        新建知识库
      </button>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-20">
      <Loader2 class="h-8 w-8 animate-spin text-cyan-400" />
      <span class="ml-3 text-slate-400">加载中...</span>
    </div>

    <div v-else-if="knowledgeBases.length === 0" class="flex flex-col items-center justify-center py-20">
      <div class="flex h-20 w-20 items-center justify-center rounded-3xl bg-cyan-400/10">
        <Database class="h-10 w-10 text-cyan-300" />
      </div>
      <p class="mt-6 text-lg font-medium text-slate-200">还没有知识库</p>
      <p class="mt-2 text-sm text-slate-400">创建你的第一个知识库开始吧</p>
      <button
        class="mt-6 flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300"
        @click="openCreate"
      >
        <Plus class="h-4 w-4" />
        新建知识库
      </button>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="kb in knowledgeBases"
        :key="kb.id"
        class="group cursor-pointer rounded-3xl border border-white/10 bg-white/[0.03] p-5 transition hover:border-cyan-400/30 hover:bg-white/[0.05]"
        @click="openKB(kb)"
      >
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-3">
            <div class="flex h-11 w-11 items-center justify-center rounded-2xl bg-cyan-400/15 text-cyan-300">
              <Database class="h-5 w-5" />
            </div>
            <h3 class="text-base font-semibold text-slate-100">{{ kb.name }}</h3>
          </div>
          <div class="flex gap-1 opacity-0 transition group-hover:opacity-100">
            <button
              class="rounded-xl p-2 text-slate-400 transition hover:bg-white/10 hover:text-slate-200"
              title="编辑"
              @click.stop="openRename(kb)"
            >
              <Edit3 class="h-4 w-4" />
            </button>
            <button
              class="rounded-xl p-2 text-slate-400 transition hover:bg-white/10 hover:text-slate-200"
              title="索引设置"
              @click.stop="openIndexEdit(kb)"
            >
              <Settings class="h-4 w-4" />
            </button>
            <button
              class="rounded-xl p-2 text-slate-400 transition hover:bg-rose-500/20 hover:text-rose-300"
              title="删除"
              @click.stop="deleteKB(kb)"
            >
              <Trash2 class="h-4 w-4" />
            </button>
          </div>
        </div>
        <p class="mt-3 text-sm text-slate-400 line-clamp-2">{{ kb.description || '暂无描述' }}</p>
        <div class="mt-4 grid grid-cols-2 gap-3">
          <div class="rounded-2xl bg-white/[0.03] p-3">
            <p class="text-xs text-slate-500">索引类型</p>
            <p class="mt-1 text-sm font-medium text-slate-200">{{ indexTypeLabel(kb.indexType) }}</p>
          </div>
          <div class="rounded-2xl bg-white/[0.03] p-3">
            <p class="text-xs text-slate-500">切片策略</p>
            <p class="mt-1 text-sm font-medium text-slate-200">{{ chunkStrategyLabel(kb.chunkStrategy) }}</p>
          </div>
          <div class="rounded-2xl bg-white/[0.03] p-3">
            <p class="text-xs text-slate-500">文档数</p>
            <p class="mt-1 text-sm font-medium text-slate-200">{{ kb.docCount }}</p>
          </div>
          <div class="rounded-2xl bg-white/[0.03] p-3">
            <p class="text-xs text-slate-500">向量数</p>
            <p class="mt-1 text-sm font-medium text-slate-200">{{ kb.vectorCount }}</p>
          </div>
        </div>
        <div class="mt-4 border-t border-white/5 pt-3 text-xs text-slate-500">
          创建于 {{ new Date(kb.createdAt).toLocaleString() }}
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="showCreate"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
        @click.self="showCreate = false"
      >
        <div class="max-h-[90vh] w-full max-w-2xl overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-xl font-semibold text-slate-100">新建知识库</h3>
            <button
              class="rounded-xl bg-white/10 p-2 text-slate-300 transition hover:bg-white/15 hover:text-white"
              @click="showCreate = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="space-y-5">
            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">知识库名称 <span class="text-rose-400">*</span></label>
              <input
                v-model="form.name"
                type="text"
                placeholder="例如：产品手册知识库"
                maxlength="128"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              />
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">描述</label>
              <textarea
                v-model="form.description"
                placeholder="知识库简介（可选）"
                rows="2"
                class="w-full resize-none rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              ></textarea>
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">Embedding 模型</label>
              <select
                v-model="form.embeddingModel"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                @change="onEmbeddingModelChange"
              >
                <option v-for="opt in embeddingModelOptions" :key="opt.model" :value="opt.model">
                  {{ embeddingModelLabels[opt.model] || opt.model }}
                </option>
              </select>
              <p class="mt-1 text-xs text-slate-500">知识库创建后模型与维度不可更改，请谨慎选择。</p>
            </div>

            <div>
              <label class="mb-2 block text-sm font-medium text-slate-200">向量维度</label>
              <select
                v-model.number="form.embeddingDim"
                class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              >
                <option v-for="dim in currentDims" :key="dim" :value="dim">{{ dim }} 维</option>
              </select>
            </div>

            <div>
              <label class="mb-3 block text-sm font-medium text-slate-200">切片策略</label>
              <div class="grid grid-cols-2 gap-3">
                <label
                  v-for="opt in chunkStrategyOptions"
                  :key="opt.value"
                  class="cursor-pointer rounded-2xl border p-4 transition"
                  :class="form.chunkStrategy === opt.value
                    ? 'border-cyan-400/50 bg-cyan-400/10 ring-2 ring-cyan-400/20'
                    : 'border-white/10 bg-white/[0.03] hover:bg-white/[0.06]'"
                >
                  <input
                    v-model="form.chunkStrategy"
                    type="radio"
                    :value="opt.value"
                    class="sr-only"
                  />
                  <div>
                    <strong class="block text-sm font-medium text-slate-100">{{ opt.label }}</strong>
                    <p class="mt-1 text-xs text-slate-400">{{ opt.description }}</p>
                  </div>
                </label>
              </div>
            </div>

            <template v-if="form.chunkStrategy === 'fixed'">
              <div>
                <label class="mb-2 block text-sm font-medium text-slate-200">切片大小（字符数）</label>
                <input
                  v-model.number="form.chunkSize"
                  type="number"
                  min="100"
                  max="5000"
                  class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-slate-200">切片重叠（字符数）</label>
                <input
                  v-model.number="form.chunkOverlap"
                  type="number"
                  min="0"
                  :max="form.chunkSize - 1"
                  class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                />
              </div>
            </template>

            <div>
              <label class="mb-3 block text-sm font-medium text-slate-200">索引类型</label>
              <div class="grid grid-cols-2 gap-3">
                <label
                  v-for="opt in indexTypeOptions"
                  :key="opt.value"
                  class="cursor-pointer rounded-2xl border p-4 transition"
                  :class="form.indexType === opt.value
                    ? 'border-cyan-400/50 bg-cyan-400/10 ring-2 ring-cyan-400/20'
                    : 'border-white/10 bg-white/[0.03] hover:bg-white/[0.06]'"
                >
                  <input
                    v-model="form.indexType"
                    type="radio"
                    :value="opt.value"
                    class="sr-only"
                  />
                  <div>
                    <strong class="block text-sm font-medium text-slate-100">{{ opt.label }}</strong>
                    <p class="mt-1 text-xs text-slate-400">{{ opt.description }}</p>
                  </div>
                </label>
              </div>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button
              class="rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-medium text-slate-300 transition hover:bg-white/10"
              @click="showCreate = false"
            >
              取消
            </button>
            <button
              class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300 disabled:opacity-60"
              :disabled="creating"
              @click="submitCreate"
            >
              <Loader2 v-if="creating" class="h-4 w-4 animate-spin" />
              {{ creating ? '创建中...' : '创建' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="showIndexEdit"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
        @click.self="showIndexEdit = false"
      >
        <div class="max-h-[90vh] w-full max-w-xl overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-xl font-semibold text-slate-100">修改索引类型 - {{ editingIndexKB?.name }}</h3>
            <button
              class="rounded-xl bg-white/10 p-2 text-slate-300 transition hover:bg-white/15 hover:text-white"
              @click="showIndexEdit = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div class="mb-5 rounded-2xl border border-amber-400/20 bg-amber-400/10 p-4 text-sm text-amber-100">
            注意：更换索引类型会删除原 Milvus collection 并重建。
          </div>

          <div>
            <label class="mb-3 block text-sm font-medium text-slate-200">索引类型</label>
            <div class="grid grid-cols-2 gap-3">
              <label
                v-for="opt in indexTypeOptions"
                :key="opt.value"
                class="cursor-pointer rounded-2xl border p-4 transition"
                :class="indexForm.indexType === opt.value
                  ? 'border-cyan-400/50 bg-cyan-400/10 ring-2 ring-cyan-400/20'
                  : 'border-white/10 bg-white/[0.03] hover:bg-white/[0.06]'"
              >
                <input
                  v-model="indexForm.indexType"
                  type="radio"
                  :value="opt.value"
                  class="sr-only"
                />
                <div>
                  <strong class="block text-sm font-medium text-slate-100">{{ opt.label }}</strong>
                  <p class="mt-1 text-xs text-slate-400">{{ opt.description }}</p>
                </div>
              </label>
            </div>
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button
              class="rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-medium text-slate-300 transition hover:bg-white/10"
              @click="showIndexEdit = false"
            >
              取消
            </button>
            <button
              class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300 disabled:opacity-60"
              :disabled="savingIndex"
              @click="submitIndexEdit"
            >
              <Loader2 v-if="savingIndex" class="h-4 w-4 animate-spin" />
              {{ savingIndex ? '保存中...' : '确定' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="showRename"
        class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
        @click.self="showRename = false"
      >
        <div class="max-h-[90vh] w-full max-w-md overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-xl font-semibold text-slate-100">修改知识库名称</h3>
            <button
              class="rounded-xl bg-white/10 p-2 text-slate-300 transition hover:bg-white/15 hover:text-white"
              @click="showRename = false"
            >
              <X class="h-5 w-5" />
            </button>
          </div>

          <div>
            <label class="mb-2 block text-sm font-medium text-slate-200">知识库名称</label>
            <input
              v-model="renameInput"
              type="text"
              maxlength="128"
              class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
              @keyup.enter="submitRename"
            />
          </div>

          <div class="mt-6 flex justify-end gap-3">
            <button
              class="rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-medium text-slate-300 transition hover:bg-white/10"
              @click="showRename = false"
            >
              取消
            </button>
            <button
              class="flex items-center gap-2 rounded-2xl bg-cyan-400 px-5 py-3 text-sm font-medium text-slate-950 shadow-lg shadow-cyan-400/20 transition hover:bg-cyan-300 disabled:opacity-60"
              :disabled="savingRename || !renameInput.trim()"
              @click="submitRename"
            >
              <Loader2 v-if="savingRename" class="h-4 w-4 animate-spin" />
              {{ savingRename ? '保存中...' : '确定' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
