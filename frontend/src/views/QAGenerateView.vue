<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Save, Sparkles } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { GeneratedQAItem, KnowledgeBase, QAGenerateTask } from '@/types/domain'

const route = useRoute()
const knowledgeBases = ref<KnowledgeBase[]>([])
const knowledgeBaseId = ref<number>(Number(route.query.kbId) || 0)
const count = ref(10)
const difficulty = ref('normal')
const instruction = ref('')
const items = ref<GeneratedQAItem[]>([])
const loading = ref(false)
const saving = ref(false)
const saved = ref(false)
const message = ref('')
const task = ref<QAGenerateTask | null>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

const selectedKnowledgeBase = computed(() => knowledgeBases.value.find((kb) => kb.id === knowledgeBaseId.value))
const progress = computed(() => task.value?.progress ?? 0)
const progressMessage = computed(() => task.value?.message ?? '')
const targetCount = computed(() => task.value?.targetCount ?? count.value)
const generatedCount = computed(() => task.value?.generatedCount ?? items.value.length)
const currentBatch = computed(() => task.value?.currentBatch ?? 0)
const totalBatches = computed(() => task.value?.totalBatches ?? 0)
const isGenerating = computed(() => loading.value || task.value?.status === 'pending' || task.value?.status === 'running')

async function loadKnowledgeBases() {
  const result = await api.knowledgeBases()
  knowledgeBases.value = result
  const queryId = Number(route.query.kbId)
  if (queryId && result.some((kb) => kb.id === queryId)) {
    knowledgeBaseId.value = queryId
  } else if (!knowledgeBaseId.value && result.length > 0) {
    knowledgeBaseId.value = result[0].id
  }
}

function keywordsText(item: GeneratedQAItem) {
  return (item.keywords ?? []).join('，')
}

function updateKeywords(item: GeneratedQAItem, value: string) {
  item.keywords = value
    .split(/[,，]/)
    .map((part) => part.trim())
    .filter(Boolean)
    .slice(0, 5)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function applyTaskResult(result: QAGenerateTask) {
  task.value = result
  if (result.items && result.items.length > 0) {
    items.value = result.items
  }
}

async function pollTask(taskId: number) {
  const result = await api.generateTask(taskId)
  applyTaskResult(result)
  if (result.status === 'completed') {
    stopPolling()
    loading.value = false
    message.value = result.items?.length
      ? `生成完成，共 ${result.items.length} 条问答`
      : '未从该知识库中生成有效问答，请检查知识库解析切片。'
  } else if (result.status === 'failed') {
    stopPolling()
    loading.value = false
    message.value = result.error || '生成失败'
  }
}

function startPolling(taskId: number) {
  stopPolling()
  void pollTask(taskId)
  pollTimer = setInterval(() => {
    void pollTask(taskId)
  }, 800)
}

async function generate() {
  if (!knowledgeBaseId.value) {
    message.value = '请先选择知识库'
    return
  }
  if (!selectedKnowledgeBase.value?.docCount || !selectedKnowledgeBase.value?.vectorCount) {
    message.value = '当前知识库暂无已解析并写入向量库的内容，请先上传并解析文档'
    return
  }
  loading.value = true
  message.value = ''
  items.value = []
  task.value = null
  saved.value = false
  try {
    const created = await api.generatePreview({
      knowledgeBaseId: knowledgeBaseId.value,
      count: count.value,
      difficulty: difficulty.value,
      instruction: instruction.value.trim(),
      overwrite: false,
    })
    applyTaskResult(created)
    startPolling(created.id)
  } catch (err) {
    loading.value = false
    message.value = err instanceof Error ? err.message : '创建生成任务失败'
  }
}

async function save() {
  if (saved.value || saving.value || !knowledgeBaseId.value || items.value.length === 0) return
  saving.value = true
  try {
    const result = await api.saveGenerated({ knowledgeBaseId: knowledgeBaseId.value, items: items.value, overwrite: false })
    saved.value = true
    message.value = `已保存 ${result.saved} 条问答`
  } catch (err) {
    message.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}

function remove(index: number) {
  items.value.splice(index, 1)
}

onMounted(loadKnowledgeBases)
onUnmounted(stopPolling)

watch(
  () => route.query.kbId,
  (value) => {
    const id = Number(value)
    if (id && knowledgeBases.value.some((kb) => kb.id === id)) {
      knowledgeBaseId.value = id
    }
  },
)
</script>

<template>
  <div class="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]">
    <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
      <h3 class="text-lg font-semibold">生成配置</h3>
      <div class="mt-6 space-y-5">
        <label class="block">
          <span class="text-sm text-slate-400">目标知识库</span>
          <select v-model="knowledgeBaseId" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" :disabled="isGenerating">
            <option :value="0">请选择知识库</option>
            <option v-for="kb in knowledgeBases" :key="kb.id" :value="kb.id">{{ kb.name }}（{{ kb.docCount }} 文档 / {{ kb.vectorCount }} 向量）</option>
          </select>
          <p class="mt-1 text-xs text-slate-500">问答将基于知识库内已解析并写入向量库的切片生成，不再直接基于单个原文文档生成。</p>
        </label>
        <div v-if="selectedKnowledgeBase" class="rounded-2xl border border-cyan-300/15 bg-cyan-300/10 p-4 text-sm leading-6 text-cyan-50">
          当前知识库：{{ selectedKnowledgeBase.name }} · {{ selectedKnowledgeBase.embeddingModel }} · {{ selectedKnowledgeBase.embeddingDim }} 维 · {{ selectedKnowledgeBase.chunkStrategy }} 切片
        </div>
        <label class="block">
          <span class="text-sm text-slate-400">生成数量</span>
          <input v-model.number="count" type="number" min="1" max="50" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" :disabled="isGenerating" />
          <p class="mt-1 text-xs text-slate-500">超过该数量将自动分批调用大模型，每批条数可在问答设置中配置</p>
        </label>
        <label class="block">
          <span class="text-sm text-slate-400">生成要求</span>
          <textarea
            v-model="instruction"
            rows="5"
            class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 text-sm leading-6 outline-none placeholder:text-slate-500"
            placeholder="可选。描述你想生成的问题方向，或直接列出指定问题，例如：&#10;1. 重点生成关于安装步骤和注意事项的问答&#10;2. 指定问题：产品保修期是多久？"
            :disabled="isGenerating"
          />
          <p class="mt-1 text-xs text-slate-500">填写后大模型会优先按你的要求生成问答；留空则自动从知识库切片中提取</p>
        </label>
        <label class="block">
          <span class="text-sm text-slate-400">问题难度</span>
          <select v-model="difficulty" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" :disabled="isGenerating">
            <option value="easy">基础识记</option>
            <option value="normal">核心理解</option>
            <option value="hard">综合分析</option>
          </select>
        </label>

        <div v-if="isGenerating" class="rounded-2xl border border-cyan-300/20 bg-cyan-300/10 p-4">
          <div class="mb-2 flex items-center justify-between text-sm">
            <span class="text-cyan-100">{{ progressMessage || '正在生成...' }}</span>
            <span class="text-cyan-200">{{ progress }}%</span>
          </div>
          <div class="h-2.5 overflow-hidden rounded-full bg-slate-950/60">
            <div class="h-full rounded-full bg-gradient-to-r from-cyan-300 to-cyan-500 transition-all duration-300" :style="{ width: `${Math.max(progress, 2)}%` }"></div>
          </div>
          <div class="mt-3 flex flex-wrap gap-3 text-xs text-slate-300">
            <span v-if="totalBatches > 0" class="rounded-full bg-white/10 px-3 py-1">批次 {{ currentBatch }}/{{ totalBatches }}</span>
            <span class="rounded-full bg-white/10 px-3 py-1">已生成 {{ generatedCount }}/{{ targetCount }} 条</span>
          </div>
        </div>

        <div class="flex gap-3">
          <button class="inline-flex flex-1 items-center justify-center gap-2 rounded-2xl bg-cyan-300 px-5 py-3 font-medium text-slate-950 disabled:opacity-60" :disabled="isGenerating" @click="generate">
            <Sparkles class="h-5 w-5" />{{ isGenerating ? '生成中...' : '生成预览' }}
          </button>
          <button
            class="inline-flex flex-1 items-center justify-center gap-2 rounded-2xl bg-amber-300 px-5 py-3 font-medium text-slate-950 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="saving || saved || items.length === 0 || isGenerating"
            @click="save"
          >
            <Save class="h-5 w-5" />{{ saved ? '已保存' : saving ? '保存中...' : '保存问答' }}
          </button>
        </div>
        <p v-if="message && !isGenerating" class="rounded-2xl border border-cyan-300/20 bg-cyan-300/10 p-4 text-sm text-cyan-100">{{ message }}</p>
      </div>
    </section>

    <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
      <div class="mb-5 flex items-center justify-between">
        <h3 class="text-lg font-semibold">生成结果预览</h3>
        <span class="text-sm text-slate-400">
          <template v-if="isGenerating">{{ generatedCount }}/{{ targetCount }} 条</template>
          <template v-else>{{ items.length }} 条</template>
        </span>
      </div>
      <div v-if="isGenerating && items.length === 0" class="rounded-2xl border border-dashed border-white/10 p-12 text-center text-slate-400">
        <p>正在调用大模型生成第 1 批...</p>
        <p class="mt-2 text-sm text-slate-500">{{ progressMessage }}</p>
      </div>
      <div v-else-if="items.length === 0" class="rounded-2xl border border-dashed border-white/10 p-12 text-center text-slate-400">生成后将在这里实时预览，可编辑后再保存。</div>
      <div v-else class="max-h-[720px] space-y-4 overflow-auto pr-2">
        <div v-if="isGenerating" class="sticky top-0 z-10 rounded-xl border border-cyan-300/20 bg-cyan-300/10 px-4 py-2 text-xs text-cyan-100">
          新问答会随每批生成实时追加显示
        </div>
        <article v-for="(item, index) in items" :key="`${index}-${item.question.slice(0, 20)}`" class="rounded-2xl border border-white/10 bg-white/[0.04] p-5">
          <div class="mb-3 flex items-center justify-between">
            <span class="text-xs text-cyan-200">#{{ index + 1 }} · 置信度 {{ Math.round(item.confidence * 100) }}%</span>
            <button class="text-sm text-rose-200 hover:text-rose-100" :disabled="isGenerating" @click="remove(index)">移除</button>
          </div>
          <input v-model="item.question" class="w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-sm font-medium outline-none" :disabled="isGenerating" />
          <textarea v-model="item.answer" rows="4" class="mt-3 w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-3 text-sm leading-6 outline-none" :disabled="isGenerating"></textarea>
          <label class="mt-3 block">
            <span class="mb-2 block text-xs text-slate-400">关键词</span>
            <input
              :value="keywordsText(item)"
              class="w-full rounded-xl border border-white/10 bg-slate-950 px-4 py-2.5 text-sm outline-none placeholder:text-slate-500"
              placeholder="多个关键词用逗号分隔"
              :disabled="isGenerating"
              @input="updateKeywords(item, ($event.target as HTMLInputElement).value)"
            />
          </label>
          <p class="mt-3 text-xs leading-5 text-slate-500">来源文档 ID：{{ item.documentId }} · 来源切片：{{ item.sourceSegmentId || '-' }}</p>
          <p class="mt-1 text-xs leading-5 text-slate-500">来源：{{ item.sourceExcerpt }}</p>
        </article>
      </div>
    </section>
  </div>
</template>
