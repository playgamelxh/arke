<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = defineProps<{
  page: number
  pageSize: number
  total: number
}>()

const emit = defineEmits<{
  change: [page: number]
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const from = computed(() => (props.total === 0 ? 0 : (props.page - 1) * props.pageSize + 1))
const to = computed(() => Math.min(props.page * props.pageSize, props.total))

function go(page: number) {
  const next = Math.min(Math.max(1, page), totalPages.value)
  if (next !== props.page) {
    emit('change', next)
  }
}
</script>

<template>
  <div v-if="total > 0" class="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm text-slate-400">
    <span>显示 {{ from }}-{{ to }} / 共 {{ total }} 条</span>
    <div class="flex items-center gap-2">
      <button
        type="button"
        class="inline-flex items-center gap-1 rounded-xl bg-white/10 px-3 py-2 hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
        :disabled="page <= 1"
        @click="go(page - 1)"
      >
        <ChevronLeft class="h-4 w-4" />上一页
      </button>
      <span class="rounded-xl bg-white/[0.04] px-3 py-2 text-slate-300">第 {{ page }} / {{ totalPages }} 页</span>
      <button
        type="button"
        class="inline-flex items-center gap-1 rounded-xl bg-white/10 px-3 py-2 hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-40"
        :disabled="page >= totalPages"
        @click="go(page + 1)"
      >
        下一页<ChevronRight class="h-4 w-4" />
      </button>
    </div>
  </div>
</template>
