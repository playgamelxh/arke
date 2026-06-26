<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Download, Edit3, Plus, Search, Sparkles, Trash2, X } from 'lucide-vue-next'
import { api } from '@/api/client'
import PaginationBar from '@/components/PaginationBar.vue'
import type { DocumentItem, QAItem } from '@/types/domain'
import { confirm } from '@/composables/useConfirm'
import { downloadCSV } from '@/utils/csv'
import { formatDate } from '@/utils/format'

const qaItems = ref<QAItem[]>([])
const documents = ref<DocumentItem[]>([])
const selectedIds = ref<number[]>([])
const keyword = ref('')
const documentId = ref('')
const enabled = ref('')
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const exporting = ref(false)
const message = ref('')
const showModal = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)
const generatingAnswer = ref(false)
const modalMessage = ref('')

const form = reactive({
  documentId: 0,
  question: '',
  answer: '',
  enabled: true,
})

const modalTitle = computed(() => (editingId.value ? '编辑问答' : '新增问答'))
const allSelected = computed({
  get: () => qaItems.value.length > 0 && selectedIds.value.length === qaItems.value.length,
  set: (value: boolean) => {
    selectedIds.value = value ? qaItems.value.map((item) => item.id) : []
  },
})

async function load(nextPage = page.value) {
  loading.value = true
  message.value = ''
  page.value = nextPage
  try {
    const [qa, docs] = await Promise.all([
      api.qaList({ page: page.value, pageSize, keyword: keyword.value, documentId: documentId.value, enabled: enabled.value }),
      api.documents({ pageSize: 200, status: 'parsed' }),
    ])
    qaItems.value = qa.list
    total.value = qa.total
    documents.value = docs.list
    selectedIds.value = selectedIds.value.filter((id) => qa.list.some((item) => item.id === id))
  } catch (err) {
    message.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function search() {
  void load(1)
}

function openCreate() {
  editingId.value = null
  form.documentId = documents.value[0]?.id || 0
  form.question = ''
  form.answer = ''
  form.enabled = true
  modalMessage.value = ''
  showModal.value = true
}

function openEdit(item: QAItem) {
  editingId.value = item.id
  form.documentId = item.documentId
  form.question = item.question
  form.answer = item.answer
  form.enabled = item.enabled
  modalMessage.value = ''
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingId.value = null
  modalMessage.value = ''
}

const canGenerateAnswer = computed(() => form.documentId > 0 && form.question.trim().length > 0)
const generateAnswerLabel = computed(() => (form.answer.trim() ? '刷新答案' : '大模型生成'))

async function generateAnswer() {
  if (!canGenerateAnswer.value || generatingAnswer.value) return
  generatingAnswer.value = true
  modalMessage.value = ''
  try {
    const result = await api.generateAnswer({
      documentId: form.documentId,
      question: form.question.trim(),
    })
    form.answer = result.answer
    modalMessage.value = '答案已生成，可继续编辑后保存'
  } catch (err) {
    modalMessage.value = err instanceof Error ? err.message : '生成答案失败'
  } finally {
    generatingAnswer.value = false
  }
}

async function save() {
  if (!form.documentId || !form.question.trim() || !form.answer.trim()) {
    message.value = '请填写文档、问题和答案'
    return
  }
  saving.value = true
  message.value = ''
  const existingTags = editingId.value ? qaItems.value.find((item) => item.id === editingId.value)?.tags ?? [] : []
  const payload = {
    documentId: form.documentId,
    question: form.question.trim(),
    answer: form.answer.trim(),
    tags: existingTags,
    enabled: form.enabled,
  }
  try {
    if (editingId.value) {
      await api.updateQA(editingId.value, payload)
      message.value = '问答已更新'
    } else {
      await api.createQA(payload)
      message.value = '问答已新增'
    }
    closeModal()
    await load()
  } catch (err) {
    message.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

async function remove(id: number) {
  const ok = await confirm({
    title: '删除问答',
    message: '确定删除该问答吗？此操作不可恢复。',
    danger: true,
  })
  if (!ok) return
  await api.deleteQA(id)
  await load()
}

async function batchRemove() {
  if (selectedIds.value.length === 0) return
  const ok = await confirm({
    title: '批量删除问答',
    message: `确定删除选中的 ${selectedIds.value.length} 条问答吗？此操作不可恢复。`,
    danger: true,
  })
  if (!ok) return
  await api.batchDeleteQA(selectedIds.value)
  selectedIds.value = []
  await load()
}

async function exportCSV() {
  if (exporting.value) return
  exporting.value = true
  message.value = ''
  const exportPageSize = 100
  const filters = {
    keyword: keyword.value,
    documentId: documentId.value,
    enabled: enabled.value,
  }
  try {
    const first = await api.qaList({ page: 1, pageSize: exportPageSize, ...filters })
    const allItems = [...first.list]
    const totalPages = Math.ceil(first.total / exportPageSize)
    for (let p = 2; p <= totalPages; p++) {
      const result = await api.qaList({ page: p, pageSize: exportPageSize, ...filters })
      allItems.push(...result.list)
    }
    const rows = allItems.map((item) => [
      item.documentName,
      item.question,
      item.answer,
      item.tags.join('，'),
    ])
    downloadCSV(`qa-export-${Date.now()}.csv`, ['所属文档', '问题', '答案', '关键词'], rows)
    message.value = `已导出 ${allItems.length} 条问答`
  } catch (err) {
    message.value = err instanceof Error ? err.message : '导出失败'
  } finally {
    exporting.value = false
  }
}

onMounted(() => load())
</script>

<template>
  <div class="space-y-6">
    <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
      <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-lg font-semibold">问答管理</h3>
          <p class="mt-1 text-sm text-slate-400">共 {{ total }} 条问答</p>
        </div>
        <div class="flex flex-wrap gap-3">
          <button class="inline-flex items-center gap-2 rounded-2xl bg-cyan-300 px-4 py-2 text-sm font-medium text-slate-950" @click="openCreate">
            <Plus class="h-4 w-4" />新增问答
          </button>
          <button class="inline-flex items-center gap-2 rounded-2xl bg-white/10 px-4 py-2 text-sm hover:bg-white/15 disabled:opacity-60" :disabled="exporting" @click="exportCSV">
            <Download class="h-4 w-4" />{{ exporting ? '导出中...' : '导出 CSV' }}
          </button>
        </div>
      </div>

      <div class="flex flex-wrap gap-3">
        <div class="flex min-w-[220px] flex-1 items-center gap-2 rounded-2xl bg-white/10 px-3 py-2">
          <Search class="h-4 w-4 text-slate-400" />
          <input v-model="keyword" class="w-full bg-transparent text-sm outline-none placeholder:text-slate-500" placeholder="搜索问题或答案" @keyup.enter="search" />
        </div>
        <select v-model="documentId" class="rounded-2xl border border-white/10 bg-slate-950 px-3 py-2 text-sm outline-none" @change="search">
          <option value="">全部文档</option>
          <option v-for="doc in documents" :key="doc.id" :value="doc.id">{{ doc.originalName }}</option>
        </select>
        <select v-model="enabled" class="rounded-2xl border border-white/10 bg-slate-950 px-3 py-2 text-sm outline-none" @change="search">
          <option value="">全部状态</option>
          <option value="true">启用</option>
          <option value="false">停用</option>
        </select>
        <button class="rounded-2xl bg-cyan-300 px-4 py-2 text-sm font-medium text-slate-950" @click="search">查询</button>
        <button v-if="selectedIds.length" class="rounded-2xl bg-rose-400/20 px-4 py-2 text-sm text-rose-100" @click="batchRemove">批量删除 {{ selectedIds.length }} 条</button>
      </div>

      <p v-if="message" class="mt-4 rounded-2xl border border-cyan-300/20 bg-cyan-300/10 p-4 text-sm text-cyan-100">{{ message }}</p>
    </section>

    <section class="overflow-hidden rounded-[2rem] border border-white/10 bg-slate-900/70">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[960px] text-left text-sm">
          <thead class="bg-white/5 text-slate-400">
            <tr>
              <th class="px-4 py-3"><input v-model="allSelected" type="checkbox" class="h-4 w-4" /></th>
              <th class="px-4 py-3">问题</th>
              <th class="px-4 py-3">答案</th>
              <th class="px-4 py-3">所属文档</th>
              <th class="px-4 py-3">关键词</th>
              <th class="px-4 py-3">创建时间</th>
              <th class="px-4 py-3 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="7" class="px-4 py-10 text-center text-slate-400">加载中...</td>
            </tr>
            <tr v-else-if="qaItems.length === 0">
              <td colspan="7" class="px-4 py-10 text-center text-slate-400">暂无问答</td>
            </tr>
            <tr v-for="item in qaItems" :key="item.id" class="border-t border-white/10 hover:bg-white/[0.03]">
              <td class="px-4 py-4 align-top"><input v-model="selectedIds" type="checkbox" :value="item.id" class="h-4 w-4" /></td>
              <td class="max-w-xs px-4 py-4 align-top font-medium text-cyan-100">{{ item.question }}</td>
              <td class="max-w-md px-4 py-4 align-top text-slate-300">
                <p class="line-clamp-3 whitespace-pre-wrap leading-6">{{ item.answer }}</p>
              </td>
              <td class="px-4 py-4 align-top text-slate-300">{{ item.documentName }}</td>
              <td class="max-w-xs px-4 py-4 align-top">
                <div v-if="item.tags.length" class="flex flex-wrap gap-1.5">
                  <span v-for="tag in item.tags" :key="tag" class="rounded-full bg-violet-400/15 px-2.5 py-1 text-xs text-violet-100">{{ tag }}</span>
                </div>
                <span v-else class="text-slate-500">-</span>
              </td>
              <td class="px-4 py-4 align-top text-slate-400">{{ formatDate(item.createdAt) }}</td>
              <td class="px-4 py-4 align-top">
                <div class="flex justify-end gap-2">
                  <button class="rounded-xl bg-white/10 p-2 hover:bg-white/15" title="编辑" @click="openEdit(item)"><Edit3 class="h-4 w-4" /></button>
                  <button class="rounded-xl bg-rose-400/15 p-2 text-rose-100 hover:bg-rose-400/25" title="删除" @click="remove(item.id)"><Trash2 class="h-4 w-4" /></button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <PaginationBar :page="page" :page-size="pageSize" :total="total" @change="load" />
    </section>

    <Teleport to="body">
      <div v-if="showModal" class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm" @click.self="closeModal">
        <div class="max-h-[90vh] w-full max-w-2xl overflow-auto rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl">
          <div class="mb-6 flex items-center justify-between">
            <h3 class="text-xl font-semibold">{{ modalTitle }}</h3>
            <button class="rounded-xl bg-white/10 p-2 hover:bg-white/15" @click="closeModal"><X class="h-5 w-5" /></button>
          </div>
          <div class="space-y-4">
            <label class="block">
              <span class="text-sm text-slate-400">所属文档</span>
              <select v-model="form.documentId" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none">
                <option :value="0">请选择文档</option>
                <option v-for="doc in documents" :key="doc.id" :value="doc.id">{{ doc.originalName }}</option>
              </select>
            </label>
            <label class="block">
              <span class="text-sm text-slate-400">问题</span>
              <input v-model="form.question" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
            </label>
            <label class="block">
              <div class="flex items-center justify-between gap-3">
                <span class="text-sm text-slate-400">答案</span>
                <button
                  type="button"
                  class="inline-flex items-center gap-2 rounded-xl bg-violet-400/15 px-3 py-1.5 text-xs font-medium text-violet-100 transition hover:bg-violet-400/25 disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="!canGenerateAnswer || generatingAnswer"
                  @click="generateAnswer"
                >
                  <Sparkles class="h-3.5 w-3.5" />
                  {{ generatingAnswer ? '生成中...' : generateAnswerLabel }}
                </button>
              </div>
              <textarea v-model="form.answer" rows="8" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 leading-7 outline-none"></textarea>
            </label>
            <p v-if="modalMessage" class="rounded-2xl border border-cyan-300/20 bg-cyan-300/10 p-3 text-sm text-cyan-100">{{ modalMessage }}</p>
            <label class="flex items-center gap-3 rounded-2xl bg-white/[0.04] p-4 text-sm text-slate-300">
              <input v-model="form.enabled" type="checkbox" class="h-4 w-4" />启用该问答
            </label>
            <div class="flex gap-3 pt-2">
              <button class="flex-1 rounded-2xl bg-cyan-300 px-5 py-3 font-medium text-slate-950 disabled:opacity-60" :disabled="saving" @click="save">{{ saving ? '保存中...' : '保存' }}</button>
              <button class="rounded-2xl bg-white/10 px-5 py-3 hover:bg-white/15" @click="closeModal">取消</button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
