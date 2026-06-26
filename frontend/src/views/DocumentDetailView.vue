<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { ArrowLeft, Check, FileText, Layers3, Pencil, Wand2, X } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { DocumentItem, DocumentSegment } from '@/types/domain'
import { formatDate, formatSize, statusClass, statusText } from '@/utils/format'
import { sanitizeParsedText } from '@/utils/text'

const route = useRoute()
const id = Number(route.params.id)
const kbId = route.query.kbId ? Number(route.query.kbId) : null
const document = ref<DocumentItem | null>(null)
const segments = ref<DocumentSegment[]>([])
const loading = ref(true)
const activeSegmentId = ref<number | null>(null)
const editingSegmentId = ref<number | null>(null)
const editTitle = ref('')
const editContent = ref('')
const saving = ref(false)
const message = ref('')

const activeSegment = computed(() => segments.value.find((item) => item.id === activeSegmentId.value) || segments.value[0] || null)
const totalChars = computed(() => segments.value.reduce((sum, item) => sum + item.content.length, 0))
const indexedCount = computed(() => segments.value.filter((item) => item.vectorId || item.indexedAt).length)
const isEditing = computed(() => editingSegmentId.value === activeSegment.value?.id)

async function load() {
  loading.value = true
  try {
    const [doc, segs] = await Promise.all([api.document(id), api.segments(id)])
    document.value = doc
    segments.value = segs
    if (!activeSegmentId.value || !segs.some((item) => item.id === activeSegmentId.value)) {
      activeSegmentId.value = segs[0]?.id ?? null
    }
  } finally {
    loading.value = false
  }
}

function selectSegment(segment: DocumentSegment) {
  if (saving.value) return
  activeSegmentId.value = segment.id
  if (editingSegmentId.value !== segment.id) {
    cancelEdit()
  }
}

function startEdit(segment: DocumentSegment) {
  activeSegmentId.value = segment.id
  editingSegmentId.value = segment.id
  editTitle.value = segment.title || `切片 #${segment.segmentIndex}`
  editContent.value = segment.content
  message.value = ''
}

function cancelEdit() {
  editingSegmentId.value = null
  editTitle.value = ''
  editContent.value = ''
}

function cleanDraft() {
  editContent.value = sanitizeParsedText(editContent.value)
}

async function saveSegment() {
  const segment = activeSegment.value
  if (!segment || !editContent.value.trim()) {
    message.value = '切片内容不能为空'
    return
  }
  saving.value = true
  message.value = ''
  try {
    const updated = await api.updateSegment(id, segment.id, {
      title: editTitle.value.trim() || segment.title,
      content: editContent.value,
    })
    segments.value = segments.value.map((item) => (item.id === updated.id ? updated : item))
    activeSegmentId.value = updated.id
    cancelEdit()
    message.value = updated.vectorId || updated.indexedAt ? '切片已保存，并已同步更新向量库' : '切片已保存；文档索引后会写入向量库'
  } catch (err) {
    message.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <RouterLink
      :to="kbId ? `/knowledge-bases/${kbId}` : '/documents'"
      class="inline-flex items-center gap-2 text-sm text-cyan-200 hover:text-cyan-100"
    >
      <ArrowLeft class="h-4 w-4" />
      {{ kbId ? '返回知识库' : '返回文档管理' }}
    </RouterLink>

    <div v-if="loading" class="rounded-3xl border border-white/10 bg-slate-900/70 p-8 text-slate-400">加载中...</div>
    <template v-else-if="document">
      <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
        <div class="flex flex-wrap items-start justify-between gap-5">
          <div>
            <div class="flex items-center gap-3">
              <FileText class="h-7 w-7 text-cyan-200" />
              <h1 class="text-2xl font-semibold">{{ document.originalName }}</h1>
            </div>
            <p class="mt-3 text-sm text-slate-400">{{ document.fileType.toUpperCase() }} · {{ formatSize(document.fileSize) }} · {{ formatDate(document.createdAt) }}</p>
          </div>
          <span class="rounded-full border px-3 py-1 text-xs" :class="statusClass(document.status)">{{ statusText(document.status) }}</span>
        </div>
        <div class="mt-5 grid gap-3 sm:grid-cols-3">
          <div class="rounded-2xl border border-white/10 bg-white/[0.03] p-4">
            <p class="text-xs text-slate-500">切片数量</p>
            <p class="mt-1 text-2xl font-semibold text-slate-100">{{ segments.length }}</p>
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/[0.03] p-4">
            <p class="text-xs text-slate-500">总字符数</p>
            <p class="mt-1 text-2xl font-semibold text-slate-100">{{ totalChars }}</p>
          </div>
          <div class="rounded-2xl border border-white/10 bg-white/[0.03] p-4">
            <p class="text-xs text-slate-500">已入库向量</p>
            <p class="mt-1 text-2xl font-semibold text-slate-100">{{ indexedCount }}</p>
          </div>
        </div>
        <p v-if="document.parseError" class="mt-4 rounded-2xl border border-rose-300/30 bg-rose-400/10 p-4 text-sm text-rose-100">{{ document.parseError }}</p>
        <p v-if="message" class="mt-4 rounded-2xl border border-cyan-300/20 bg-cyan-300/10 p-4 text-sm text-cyan-100">{{ message }}</p>
      </section>

      <section>
        <div class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
          <div class="mb-5 flex flex-wrap items-center justify-between gap-4">
            <div>
              <h3 class="flex items-center gap-2 text-lg font-semibold">
                <Layers3 class="h-5 w-5 text-cyan-200" />
                解析切片
              </h3>
              <p class="mt-1 text-xs text-slate-500">查看每个切片的标题、内容和向量状态；编辑已索引切片后会重新写入向量库</p>
            </div>
            <button
              v-if="activeSegment && !isEditing"
              class="inline-flex items-center gap-2 rounded-2xl bg-cyan-300 px-4 py-2 text-sm font-medium text-slate-950 transition hover:bg-cyan-200"
              @click="startEdit(activeSegment)"
            >
              <Pencil class="h-4 w-4" />
              编辑当前切片
            </button>
          </div>

          <div v-if="segments.length === 0" class="rounded-2xl border border-dashed border-white/10 p-8 text-center text-slate-400">暂无切片信息，请先解析文档</div>
          <div v-else class="grid gap-5 lg:grid-cols-[20rem_1fr]">
            <div class="max-h-[760px] space-y-2 overflow-auto rounded-3xl border border-white/10 bg-slate-950/50 p-2">
              <button
                v-for="segment in segments"
                :key="segment.id"
                class="w-full rounded-2xl border p-4 text-left transition"
                :class="activeSegment?.id === segment.id
                  ? 'border-cyan-300/50 bg-cyan-300/10 shadow-lg shadow-cyan-950/20'
                  : 'border-white/5 bg-white/[0.03] hover:border-white/15 hover:bg-white/[0.06]'"
                @click="selectSegment(segment)"
              >
                <div class="flex items-center justify-between gap-3">
                  <span class="text-sm font-semibold text-slate-100">#{{ segment.segmentIndex }}</span>
                  <span
                    class="rounded-full px-2 py-0.5 text-[11px]"
                    :class="segment.vectorId || segment.indexedAt ? 'bg-emerald-300/10 text-emerald-200' : 'bg-amber-300/10 text-amber-200'"
                  >
                    {{ segment.vectorId || segment.indexedAt ? '已入库' : '未入库' }}
                  </span>
                </div>
                <p class="mt-2 line-clamp-2 text-sm text-slate-300">{{ segment.title || '未命名切片' }}</p>
                <p class="mt-2 text-xs text-slate-500">{{ segment.content.length }} 字符 · {{ segment.segmentType }}</p>
              </button>
            </div>

            <article v-if="activeSegment" class="rounded-3xl border border-white/10 bg-white/[0.04] p-5">
              <template v-if="isEditing">
                <div class="space-y-4">
                  <div>
                    <label class="mb-2 block text-xs font-medium text-slate-400">切片标题</label>
                    <input
                      v-model="editTitle"
                      class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                    />
                  </div>
                  <div>
                    <label class="mb-2 block text-xs font-medium text-slate-400">切片内容</label>
                    <textarea
                      v-model="editContent"
                      rows="22"
                      class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm leading-7 text-slate-100 outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20"
                    />
                  </div>
                  <div class="flex flex-wrap justify-end gap-3">
                    <button class="inline-flex items-center gap-2 rounded-2xl bg-white/10 px-4 py-2 text-sm text-slate-200 transition hover:bg-white/15" @click="cleanDraft">
                      <Wand2 class="h-4 w-4" />
                      清理格式
                    </button>
                    <button class="inline-flex items-center gap-2 rounded-2xl bg-white/10 px-4 py-2 text-sm text-slate-200 transition hover:bg-white/15" @click="cancelEdit">
                      <X class="h-4 w-4" />
                      取消
                    </button>
                    <button class="inline-flex items-center gap-2 rounded-2xl bg-cyan-300 px-4 py-2 text-sm font-medium text-slate-950 transition hover:bg-cyan-200 disabled:opacity-60" :disabled="saving" @click="saveSegment">
                      <Check class="h-4 w-4" />
                      {{ saving ? '保存中...' : '保存并更新向量' }}
                    </button>
                  </div>
                </div>
              </template>
              <template v-else>
                <div class="mb-4 flex flex-wrap items-start justify-between gap-4 border-b border-white/10 pb-4">
                  <div>
                    <p class="text-xs text-slate-500">切片 #{{ activeSegment.segmentIndex }}</p>
                    <h4 class="mt-1 text-xl font-semibold text-slate-100">{{ activeSegment.title || '未命名切片' }}</h4>
                  </div>
                  <div class="text-right text-xs text-slate-500">
                    <p>{{ activeSegment.content.length }} 字符</p>
                    <p v-if="activeSegment.indexedAt">向量更新：{{ formatDate(activeSegment.indexedAt) }}</p>
                  </div>
                </div>
                <p class="max-h-[640px] overflow-auto whitespace-pre-wrap text-sm leading-7 text-slate-300">{{ activeSegment.content }}</p>
              </template>
            </article>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>
