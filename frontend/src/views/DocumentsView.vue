<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { RefreshCw, Search, Sparkles, Trash2, UploadCloud } from 'lucide-vue-next'
import { api } from '@/api/client'
import PaginationBar from '@/components/PaginationBar.vue'
import type { DocumentItem } from '@/types/domain'
import { confirm } from '@/composables/useConfirm'
import { formatDate, formatSize, statusClass, statusText } from '@/utils/format'

const documents = ref<DocumentItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const keyword = ref('')
const status = ref('')
const loading = ref(false)
const uploading = ref(false)
const error = ref('')

async function load(nextPage = page.value) {
  loading.value = true
  error.value = ''
  page.value = nextPage
  try {
    const result = await api.documents({ page: page.value, pageSize, keyword: keyword.value, status: status.value })
    documents.value = result.list
    total.value = result.total
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function search() {
  void load(1)
}

async function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploading.value = true
  error.value = ''
  try {
    await api.uploadDocument(file)
    await load(1)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上传失败'
  } finally {
    uploading.value = false
    input.value = ''
  }
}

async function reparse(id: number) {
  await api.reparseDocument(id)
  await load()
}

async function remove(id: number) {
  const ok = await confirm({
    title: '删除文档',
    message: '确定删除该文档及其解析内容和关联问答吗？此操作不可恢复。',
    danger: true,
  })
  if (!ok) return
  await api.deleteDocument(id)
  await load()
}

onMounted(() => load())
</script>

<template>
  <div class="space-y-6">
    <section class="grid gap-5 xl:grid-cols-[0.8fr_1.2fr]">
      <label class="flex min-h-56 cursor-pointer flex-col items-center justify-center rounded-[2rem] border border-dashed border-cyan-300/40 bg-cyan-300/10 p-8 text-center transition hover:bg-cyan-300/15">
        <UploadCloud class="h-12 w-12 text-cyan-200" />
        <p class="mt-4 text-lg font-semibold">上传并识别文档</p>
        <p class="mt-2 text-sm text-cyan-50/70">支持 PDF、PPT、PPTX、XLS、XLSX、PNG、JPG 等，通过 MinerU 智能解析</p>
        <p class="mt-4 rounded-full bg-slate-950/40 px-4 py-2 text-sm text-cyan-100">{{ uploading ? '上传解析中...' : '选择文件' }}</p>
        <input class="hidden" type="file" accept=".pdf,.ppt,.pptx,.xls,.xlsx,.png,.jpg,.jpeg,.webp,.doc,.docx" :disabled="uploading" @change="onFileChange" />
      </label>
      <div class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
        <h3 class="text-lg font-semibold">管理说明</h3>
        <p class="mt-3 text-sm leading-7 text-slate-300">上传后系统会保存原始文件到 uploads 目录，并调用 MinerU 服务进行高精度解析。解析成功的文档可进入「生成问答」页生成问答；失败时可查看原因并重新解析。</p>
        <div v-if="error" class="mt-5 rounded-2xl border border-rose-300/30 bg-rose-400/10 p-4 text-sm text-rose-100">{{ error }}</div>
      </div>
    </section>

    <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
      <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-lg font-semibold">文档列表</h3>
          <p class="mt-1 text-sm text-slate-400">共 {{ total }} 个文档</p>
        </div>
        <div class="flex gap-3">
          <div class="flex items-center gap-2 rounded-2xl bg-white/10 px-3 py-2">
            <Search class="h-4 w-4 text-slate-400" />
            <input v-model="keyword" class="bg-transparent text-sm outline-none placeholder:text-slate-500" placeholder="搜索文件名" @keyup.enter="search" />
          </div>
          <select v-model="status" class="rounded-2xl border border-white/10 bg-slate-950 px-3 py-2 text-sm outline-none" @change="search">
            <option value="">全部状态</option>
            <option value="parsed">已解析</option>
            <option value="failed">解析失败</option>
            <option value="parsing">解析中</option>
          </select>
          <button class="rounded-2xl bg-cyan-300 px-4 py-2 text-sm font-medium text-slate-950" @click="search">查询</button>
        </div>
      </div>

      <div class="overflow-hidden rounded-2xl border border-white/10">
        <table class="w-full text-left text-sm">
          <thead class="bg-white/5 text-slate-400">
            <tr>
              <th class="px-4 py-3">文档</th>
              <th class="px-4 py-3">类型/大小</th>
              <th class="px-4 py-3">状态</th>
              <th class="px-4 py-3">解析/问答</th>
              <th class="px-4 py-3">上传时间</th>
              <th class="px-4 py-3 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="6" class="px-4 py-10 text-center text-slate-400">加载中...</td>
            </tr>
            <tr v-else-if="documents.length === 0">
              <td colspan="6" class="px-4 py-10 text-center text-slate-400">暂无文档</td>
            </tr>
            <tr v-for="doc in documents" :key="doc.id" class="border-t border-white/10 hover:bg-white/[0.03]">
              <td class="px-4 py-4">
                <RouterLink :to="`/documents/${doc.id}`" class="font-medium text-cyan-100 hover:text-cyan-200">{{ doc.originalName }}</RouterLink>
                <p v-if="doc.parseError" class="mt-1 text-xs text-rose-200">{{ doc.parseError }}</p>
              </td>
              <td class="px-4 py-4 text-slate-300">{{ doc.fileType.toUpperCase() }} · {{ formatSize(doc.fileSize) }}</td>
              <td class="px-4 py-4"><span class="rounded-full border px-3 py-1 text-xs" :class="statusClass(doc.status)">{{ statusText(doc.status) }}</span></td>
              <td class="px-4 py-4 text-slate-300">{{ doc.segmentCount > 0 ? '已识别' : '-' }} / {{ doc.qaCount }} 问答</td>
              <td class="px-4 py-4 text-slate-400">{{ formatDate(doc.createdAt) }}</td>
              <td class="px-4 py-4">
                <div class="flex justify-end gap-2">
                  <RouterLink
                    v-if="doc.status === 'parsed'"
                    :to="doc.knowledgeBaseId ? `/qa-generate?kbId=${doc.knowledgeBaseId}` : '/qa-generate'"
                    class="inline-flex items-center gap-1 rounded-xl bg-cyan-300/15 px-3 py-2 text-xs font-medium text-cyan-100 hover:bg-cyan-300/25"
                    title="生成问答"
                  >
                    <Sparkles class="h-4 w-4" />生成问答
                  </RouterLink>
                  <button class="rounded-xl bg-white/10 p-2 hover:bg-white/15" title="重新解析" @click="reparse(doc.id)"><RefreshCw class="h-4 w-4" /></button>
                  <button class="rounded-xl bg-rose-400/15 p-2 text-rose-100 hover:bg-rose-400/25" title="删除" @click="remove(doc.id)"><Trash2 class="h-4 w-4" /></button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <PaginationBar :page="page" :page-size="pageSize" :total="total" @change="load" />
    </section>
  </div>
</template>
