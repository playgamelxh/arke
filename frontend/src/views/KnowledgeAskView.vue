<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Bot, Database, Loader2, MessageSquareText, RotateCcw, Search, SlidersHorizontal } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { KnowledgeAskResult, KnowledgeBase } from '@/types/domain'

const knowledgeBases = ref<KnowledgeBase[]>([])
const knowledgeBaseId = ref(0)
const question = ref('')
const recallCount = ref(10)
const useCount = ref(5)
const rerankMode = ref('similarity')
const loading = ref(false)
const message = ref('')
const result = ref<KnowledgeAskResult | null>(null)

const selectedKnowledgeBase = computed(() => knowledgeBases.value.find((kb) => kb.id === knowledgeBaseId.value))
const canAsk = computed(() => Boolean(knowledgeBaseId.value && question.value.trim() && !loading.value))

const rerankOptions = [
  { value: 'similarity', label: '相似度优先', desc: '按向量召回相似度从高到低使用' },
  { value: 'keyword', label: '关键词命中优先', desc: '按问题关键词在召回内容中的命中数量重排' },
  { value: 'length', label: '长内容优先', desc: '优先使用信息量更完整的长切片' },
]

const examples = ['这个知识库主要讲了什么？', '请总结核心流程和注意事项', '有哪些关键概念需要重点理解？']

async function loadKnowledgeBases() {
  knowledgeBases.value = await api.knowledgeBases()
  if (!knowledgeBaseId.value && knowledgeBases.value.length > 0) {
    knowledgeBaseId.value = knowledgeBases.value[0].id
  }
}

async function ask() {
  if (!knowledgeBaseId.value) {
    message.value = '请先选择知识库'
    return
  }
  if (!question.value.trim()) {
    message.value = '请输入问题'
    return
  }
  if (!selectedKnowledgeBase.value?.vectorCount) {
    message.value = '当前知识库暂无可召回向量，请先上传并解析文档'
    return
  }
  if (recallCount.value < 1 || recallCount.value > 50) {
    message.value = '召回数量需在 1-50 之间'
    return
  }
  if (useCount.value < 1 || useCount.value > recallCount.value) {
    message.value = '使用数量需大于 0 且不能超过召回数量'
    return
  }
  loading.value = true
  message.value = ''
  result.value = null
  try {
    result.value = await api.knowledgeAsk({
      knowledgeBaseId: knowledgeBaseId.value,
      question: question.value.trim(),
      recallCount: recallCount.value,
      rerankMode: rerankMode.value,
      useCount: useCount.value,
    })
  } catch (err) {
    message.value = err instanceof Error ? err.message : '生成回答失败'
  } finally {
    loading.value = false
  }
}

function reset() {
  question.value = ''
  result.value = null
  message.value = ''
}

function useExample(text: string) {
  question.value = text
}

watch(recallCount, (value) => {
  if (useCount.value > value) {
    useCount.value = value
  }
})

onMounted(loadKnowledgeBases)
</script>

<template>
  <div class="grid gap-6 xl:grid-cols-[0.82fr_1.18fr]">
    <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6 shadow-2xl shadow-black/10">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="text-sm text-cyan-200">Knowledge Chat</p>
          <h3 class="mt-1 text-xl font-semibold">知识问答</h3>
          <p class="mt-2 text-sm leading-7 text-slate-400">选择知识库后配置召回、重排和最终使用数量，再基于召回内容调用大模型回答。</p>
        </div>
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-300/10 text-cyan-200">
          <MessageSquareText class="h-6 w-6" />
        </div>
      </div>

      <div class="mt-7 space-y-5">
        <label class="block">
          <span class="mb-2 flex items-center gap-2 text-sm text-slate-300"><Database class="h-4 w-4 text-cyan-200" />目标知识库</span>
          <select v-model="knowledgeBaseId" class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm outline-none transition focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20" :disabled="loading">
            <option :value="0">请选择知识库</option>
            <option v-for="kb in knowledgeBases" :key="kb.id" :value="kb.id">{{ kb.name }}（{{ kb.docCount }} 文档 / {{ kb.vectorCount }} 向量）</option>
          </select>
        </label>

        <div v-if="selectedKnowledgeBase" class="rounded-2xl border border-cyan-300/15 bg-cyan-300/10 p-4 text-sm leading-6 text-cyan-50">
          {{ selectedKnowledgeBase.name }} · {{ selectedKnowledgeBase.embeddingModel }} · {{ selectedKnowledgeBase.embeddingDim }} 维 · {{ selectedKnowledgeBase.vectorCount }} 向量
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <label class="block">
            <span class="mb-2 flex items-center gap-2 text-sm text-slate-300"><Search class="h-4 w-4 text-cyan-200" />召回数量</span>
            <input v-model.number="recallCount" type="number" min="1" max="50" class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm outline-none" :disabled="loading" />
            <p class="mt-1 text-xs text-slate-500">先从向量库召回的切片数量</p>
          </label>
          <label class="block">
            <span class="mb-2 flex items-center gap-2 text-sm text-slate-300"><SlidersHorizontal class="h-4 w-4 text-cyan-200" />使用数量</span>
            <input v-model.number="useCount" type="number" min="1" :max="recallCount" class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm outline-none" :disabled="loading" />
            <p class="mt-1 text-xs text-slate-500">重排后送给大模型的切片数量</p>
          </label>
        </div>

        <label class="block">
          <span class="mb-2 block text-sm text-slate-300">重排方式</span>
          <div class="grid gap-3">
            <button
              v-for="option in rerankOptions"
              :key="option.value"
              type="button"
              class="rounded-2xl border px-4 py-3 text-left transition"
              :class="rerankMode === option.value ? 'border-cyan-300/40 bg-cyan-300/10 text-cyan-50' : 'border-white/10 bg-slate-950/70 text-slate-300 hover:bg-white/5'"
              :disabled="loading"
              @click="rerankMode = option.value"
            >
              <span class="block text-sm font-medium">{{ option.label }}</span>
              <span class="mt-1 block text-xs text-slate-500">{{ option.desc }}</span>
            </button>
          </div>
        </label>

        <label class="block">
          <span class="mb-2 block text-sm text-slate-300">用户问题</span>
          <textarea v-model="question" rows="5" class="w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm leading-6 outline-none placeholder:text-slate-500 focus:border-cyan-400/50 focus:ring-2 focus:ring-cyan-400/20" placeholder="请输入要基于知识库回答的问题" :disabled="loading" />
        </label>

        <div class="flex flex-wrap gap-2">
          <button v-for="example in examples" :key="example" type="button" class="rounded-full border border-white/10 bg-white/5 px-3 py-1.5 text-xs text-slate-300 hover:bg-white/10" :disabled="loading" @click="useExample(example)">{{ example }}</button>
        </div>

        <div class="flex gap-3">
          <button class="inline-flex flex-1 items-center justify-center gap-2 rounded-2xl bg-cyan-300 px-5 py-3 font-medium text-slate-950 disabled:cursor-not-allowed disabled:opacity-60" :disabled="!canAsk" @click="ask">
            <Loader2 v-if="loading" class="h-5 w-5 animate-spin" />
            <Bot v-else class="h-5 w-5" />
            {{ loading ? '生成中...' : '开始提问' }}
          </button>
          <button class="inline-flex items-center justify-center gap-2 rounded-2xl border border-white/10 px-5 py-3 text-slate-300 hover:bg-white/5" :disabled="loading" @click="reset">
            <RotateCcw class="h-5 w-5" />重置
          </button>
        </div>
        <p v-if="message" class="rounded-2xl border border-rose-300/20 bg-rose-300/10 p-4 text-sm text-rose-100">{{ message }}</p>
      </div>
    </section>

    <section class="space-y-6">
      <div class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6 shadow-2xl shadow-black/10">
        <div class="mb-5 flex items-center justify-between">
          <h3 class="text-lg font-semibold">大模型回答</h3>
          <span v-if="result" class="rounded-full border border-cyan-300/20 bg-cyan-300/10 px-3 py-1 text-xs text-cyan-100">置信度 {{ Math.round(result.confidence * 100) }}%</span>
        </div>
        <div v-if="loading" class="rounded-2xl border border-dashed border-cyan-300/20 bg-cyan-300/5 p-12 text-center text-cyan-100">
          <Loader2 class="mx-auto h-8 w-8 animate-spin" />
          <p class="mt-4">正在召回知识库内容并生成答案...</p>
        </div>
        <div v-else-if="!result" class="rounded-2xl border border-dashed border-white/10 p-12 text-center text-slate-400">提问后将在这里展示基于召回内容生成的答案。</div>
        <div v-else class="space-y-5">
          <div class="rounded-2xl bg-slate-950/70 p-5 text-sm leading-7 text-slate-100 whitespace-pre-wrap">{{ result.answer }}</div>
          <div class="rounded-2xl border border-amber-300/20 bg-amber-300/10 p-4 text-sm leading-6 text-amber-50">
            <p class="font-medium">关键来源摘录</p>
            <p class="mt-2 text-amber-50/80">{{ result.sourceExcerpt || '模型未返回来源摘录' }}</p>
          </div>
          <div class="grid gap-3 text-sm text-slate-300 md:grid-cols-3">
            <div class="rounded-2xl bg-white/[0.04] p-4">召回数量：{{ result.recallCount }}</div>
            <div class="rounded-2xl bg-white/[0.04] p-4">使用数量：{{ result.useCount }}</div>
            <div class="rounded-2xl bg-white/[0.04] p-4">重排方式：{{ rerankOptions.find((item) => item.value === result?.rerankMode)?.label || result.rerankMode }}</div>
          </div>
        </div>
      </div>

      <div class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
        <div class="mb-5 flex items-center justify-between">
          <h3 class="text-lg font-semibold">使用的召回内容</h3>
          <span class="text-sm text-slate-400">{{ result?.sources.length || 0 }} 条</span>
        </div>
        <div v-if="!result" class="rounded-2xl border border-dashed border-white/10 p-10 text-center text-slate-400">暂无召回内容。</div>
        <div v-else class="max-h-[520px] space-y-3 overflow-auto pr-2">
          <article v-for="(source, index) in result.sources" :key="`${source.documentId}-${source.sourceSegmentId}-${index}`" class="rounded-2xl border border-white/10 bg-white/[0.04] p-4">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-slate-400">
              <span class="rounded-full bg-cyan-300/10 px-3 py-1 text-cyan-100">#{{ index + 1 }} · 相似度 {{ Math.round(source.score * 10000) / 100 }}%</span>
              <span>文档 ID：{{ source.documentId }} · 切片 ID：{{ source.sourceSegmentId || '-' }}</span>
            </div>
            <p class="text-sm leading-7 text-slate-300">{{ source.content }}</p>
          </article>
        </div>
      </div>
    </section>
  </div>
</template>
