<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Activity, CheckCircle2, Database, FileText, HelpCircle, Layers3, XCircle } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { KnowledgeBase, Stats } from '@/types/domain'
import { formatDate, statusClass, statusText } from '@/utils/format'

const stats = ref<Stats>({ documents: 0, parsed: 0, failed: 0, qa: 0, recentDocuments: [] })
const knowledgeBases = ref<KnowledgeBase[]>([])
const loading = ref(true)

const totalVectors = computed(() => knowledgeBases.value.reduce((sum, kb) => sum + (kb.vectorCount || 0), 0))
const totalKBDocuments = computed(() => knowledgeBases.value.reduce((sum, kb) => sum + (kb.docCount || 0), 0))
const recentKnowledgeBases = computed(() => [...knowledgeBases.value].sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()).slice(0, 5))

const cards = computed(() => [
  { key: 'knowledgeBases', label: '知识库数量', value: knowledgeBases.value.length, icon: Database, accent: 'text-cyan-200' },
  { key: 'documents', label: '文档总数', value: stats.value.documents, icon: FileText, accent: 'text-sky-200' },
  { key: 'parsed', label: '解析成功', value: stats.value.parsed, icon: CheckCircle2, accent: 'text-emerald-200' },
  { key: 'qa', label: '问答总数', value: stats.value.qa, icon: HelpCircle, accent: 'text-amber-200' },
  { key: 'vectors', label: '向量总数', value: totalVectors.value, icon: Layers3, accent: 'text-violet-200' },
  { key: 'failed', label: '解析失败', value: stats.value.failed, icon: XCircle, accent: 'text-rose-200' },
])

async function load() {
  loading.value = true
  try {
    const [statsResult, kbResult] = await Promise.all([api.stats(), api.knowledgeBases()])
    stats.value = statsResult
    knowledgeBases.value = kbResult
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-8">
    <section class="overflow-hidden rounded-[2rem] border border-white/10 bg-white/[0.06] p-8 shadow-2xl shadow-black/20">
      <div class="flex items-start justify-between gap-8">
        <div>
          <p class="text-sm text-cyan-100/70">知识库、文档、问答统一运营看板</p>
          <h1 class="mt-3 max-w-3xl text-4xl font-semibold leading-tight">把企业资料沉淀为可检索、可生成、可维护的知识资产</h1>
          <p class="mt-4 max-w-2xl text-sm leading-7 text-slate-300">看板聚合知识库、文档解析、向量写入和问答生成数据，用于快速判断当前知识资产规模与处理状态。</p>
        </div>
        <div class="hidden rounded-full border border-cyan-300/20 bg-cyan-300/10 px-5 py-3 text-cyan-100 lg:block">Knowledge Ops</div>
      </div>
    </section>

    <section class="grid gap-5 md:grid-cols-2 xl:grid-cols-6">
      <div v-for="card in cards" :key="card.key" class="rounded-3xl border border-white/10 bg-slate-900/70 p-6 transition hover:-translate-y-1 hover:bg-slate-900">
        <div class="flex items-center justify-between">
          <span class="text-sm text-slate-400">{{ card.label }}</span>
          <component :is="card.icon" class="h-5 w-5" :class="card.accent" />
        </div>
        <div class="mt-5 text-4xl font-semibold">{{ card.value }}</div>
      </div>
    </section>

    <section class="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
      <div class="rounded-3xl border border-white/10 bg-slate-900/70 p-6">
        <div class="mb-5 flex items-center justify-between">
          <div>
            <h3 class="text-lg font-semibold">知识库概览</h3>
            <p class="mt-1 text-sm text-slate-400">共 {{ knowledgeBases.length }} 个知识库，关联 {{ totalKBDocuments }} 个文档，写入 {{ totalVectors }} 个向量</p>
          </div>
          <Database class="h-5 w-5 text-cyan-200" />
        </div>
        <div v-if="loading" class="space-y-3">
          <div v-for="i in 4" :key="i" class="h-20 animate-pulse rounded-2xl bg-white/10"></div>
        </div>
        <div v-else-if="recentKnowledgeBases.length === 0" class="rounded-2xl border border-dashed border-white/10 p-8 text-center text-slate-400">暂无知识库，请先创建知识库。</div>
        <div v-else class="space-y-3">
          <RouterLink v-for="kb in recentKnowledgeBases" :key="kb.id" :to="`/knowledge-bases/${kb.id}`" class="block rounded-2xl bg-white/[0.04] p-4 transition hover:bg-white/[0.07]">
            <div class="flex items-start justify-between gap-4">
              <div>
                <p class="font-medium text-slate-100">{{ kb.name }}</p>
                <p class="mt-1 text-xs text-slate-400">{{ kb.embeddingModel }} · {{ kb.embeddingDim }} 维 · {{ kb.indexType }}</p>
              </div>
              <span class="rounded-full border border-cyan-300/20 bg-cyan-300/10 px-3 py-1 text-xs text-cyan-100">{{ kb.docCount }} 文档</span>
            </div>
            <div class="mt-3 grid grid-cols-2 gap-3 text-xs text-slate-400">
              <span>向量数：{{ kb.vectorCount }}</span>
              <span>创建时间：{{ formatDate(kb.createdAt) }}</span>
            </div>
          </RouterLink>
        </div>
      </div>

      <div class="rounded-3xl border border-white/10 bg-slate-900/70 p-6">
        <div class="mb-5 flex items-center justify-between">
          <h3 class="text-lg font-semibold">最近上传文档</h3>
          <Activity class="h-5 w-5 text-cyan-200" />
        </div>
        <div v-if="loading" class="space-y-3">
          <div v-for="i in 4" :key="i" class="h-16 animate-pulse rounded-2xl bg-white/10"></div>
        </div>
        <div v-else-if="stats.recentDocuments.length === 0" class="rounded-2xl border border-dashed border-white/10 p-8 text-center text-slate-400">暂无文档，请先在知识库中上传。</div>
        <div v-else class="space-y-3">
          <RouterLink v-for="doc in stats.recentDocuments" :key="doc.id" :to="`/documents/${doc.id}`" class="flex items-center justify-between rounded-2xl bg-white/[0.04] p-4 transition hover:bg-white/[0.07]">
            <div>
              <p class="font-medium text-slate-100">{{ doc.originalName }}</p>
              <p class="mt-1 text-xs text-slate-400">{{ formatDate(doc.createdAt) }} · {{ doc.qaCount }} 个问答</p>
            </div>
            <span class="rounded-full border px-3 py-1 text-xs" :class="statusClass(doc.status)">{{ statusText(doc.status) }}</span>
          </RouterLink>
        </div>
      </div>
    </section>

    <section class="rounded-3xl border border-cyan-300/20 bg-cyan-300/10 p-6">
      <h3 class="text-lg font-semibold text-cyan-50">推荐操作路径</h3>
      <ol class="mt-5 grid gap-4 text-sm text-cyan-50/80 md:grid-cols-2 xl:grid-cols-4">
        <li class="rounded-2xl bg-slate-950/30 p-4">1. 创建知识库并选择切片策略、Embedding 模型和索引类型。</li>
        <li class="rounded-2xl bg-slate-950/30 p-4">2. 在知识库中上传文档，等待系统解析并写入向量库。</li>
        <li class="rounded-2xl bg-slate-950/30 p-4">3. 进入生成问答页选择文档并生成问答预览。</li>
        <li class="rounded-2xl bg-slate-950/30 p-4">4. 保存后在问答管理页编辑、启用或删除。</li>
      </ol>
    </section>
  </div>
</template>
