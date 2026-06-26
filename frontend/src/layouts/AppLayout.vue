<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, RouterLink, RouterView } from 'vue-router'
import { BrainCircuit, ChevronDown, Database, HelpCircle, LayoutDashboard, Settings, SlidersHorizontal, Sparkles } from 'lucide-vue-next'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import Toast from '@/components/Toast.vue'

const route = useRoute()

const primaryNavItems = [
  { path: '/dashboard', label: '数据看板', icon: LayoutDashboard },
  { path: '/knowledge-bases', label: '知识库', icon: Database },
  { path: '/knowledge-ask', label: '知识问答', icon: HelpCircle },
]

const qaNavItems = [
  { path: '/qa-generate', label: '生成问答', icon: Sparkles },
  { path: '/qa', label: '问答管理', icon: HelpCircle, exact: true },
  { path: '/qa/settings', label: '问答设置', icon: SlidersHorizontal },
]

const qaExpanded = ref(route.path.startsWith('/qa'))
const qaActive = computed(() => route.path.startsWith('/qa-generate') || route.path.startsWith('/qa'))

watch(
  () => route.path,
  (path) => {
    if (path.startsWith('/qa-generate') || path.startsWith('/qa')) {
      qaExpanded.value = true
    }
  },
)

const isActiveNav = (item: { path: string; exact?: boolean }) => item.exact ? route.path === item.path : route.path.startsWith(item.path)

const currentTitle = computed(() => {
  if (route.path.startsWith('/dashboard')) return '数据看板'
  if (route.path.startsWith('/knowledge-bases')) return '知识库'
  if (route.path.startsWith('/knowledge-ask')) return '知识问答'
  if (route.path.startsWith('/qa-generate')) return '生成问答'
  if (route.path.startsWith('/qa/settings')) return '问答设置'
  if (route.path === '/qa') return '问答管理'
  if (route.path.startsWith('/settings')) return '系统设置'
  return '知识库管理'
})
</script>

<template>
  <div class="min-h-screen bg-[#08111f] text-slate-100">
    <div class="fixed inset-0 pointer-events-none bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.2),transparent_32%),radial-gradient(circle_at_70%_20%,rgba(245,158,11,0.16),transparent_24%),linear-gradient(135deg,#08111f_0%,#0f172a_48%,#111827_100%)]"></div>
    <div class="relative flex min-h-screen">
      <aside class="w-72 border-r border-white/10 bg-slate-950/60 px-5 py-6 backdrop-blur-xl">
        <div class="mb-8 flex items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-400 text-slate-950 shadow-lg shadow-cyan-400/20">
            <BrainCircuit class="h-7 w-7" />
          </div>
          <div>
            <p class="text-sm text-cyan-200">Knowledge Base</p>
            <h1 class="text-lg font-semibold tracking-wide">知识库管理</h1>
          </div>
        </div>
        <nav class="space-y-2">
          <RouterLink
            v-for="item in primaryNavItems"
            :key="item.path"
            :to="item.path"
            class="group flex items-center gap-3 rounded-2xl px-4 py-3 text-sm text-slate-300 transition hover:bg-white/10 hover:text-white"
            :class="isActiveNav(item) ? 'bg-cyan-400/15 text-cyan-100 ring-1 ring-cyan-300/20' : ''"
          >
            <component :is="item.icon" class="h-5 w-5" />
            <span>{{ item.label }}</span>
          </RouterLink>

          <button
            type="button"
            class="group flex w-full items-center gap-3 rounded-2xl px-4 py-3 text-sm text-slate-300 transition hover:bg-white/10 hover:text-white"
            :class="qaActive ? 'bg-cyan-400/15 text-cyan-100 ring-1 ring-cyan-300/20' : ''"
            @click="qaExpanded = !qaExpanded"
          >
            <HelpCircle class="h-5 w-5" />
            <span class="flex-1 text-left">问答生成</span>
            <ChevronDown class="h-4 w-4 transition-transform" :class="qaExpanded ? 'rotate-180' : ''" />
          </button>

          <div v-if="qaExpanded" class="space-y-2 pl-4">
            <RouterLink
              v-for="item in qaNavItems"
              :key="item.path"
              :to="item.path"
              class="group flex items-center gap-3 rounded-2xl px-4 py-3 text-sm text-slate-300 transition hover:bg-white/10 hover:text-white"
              :class="isActiveNav(item) ? 'bg-cyan-400/15 text-cyan-100 ring-1 ring-cyan-300/20' : ''"
            >
              <component :is="item.icon" class="h-5 w-5" />
              <span>{{ item.label }}</span>
            </RouterLink>
          </div>

          <RouterLink
            to="/settings"
            class="group flex items-center gap-3 rounded-2xl px-4 py-3 text-sm text-slate-300 transition hover:bg-white/10 hover:text-white"
            :class="route.path.startsWith('/settings') ? 'bg-cyan-400/15 text-cyan-100 ring-1 ring-cyan-300/20' : ''"
          >
            <Settings class="h-5 w-5" />
            <span>系统设置</span>
          </RouterLink>
        </nav>
        <div class="mt-8 rounded-3xl border border-cyan-300/20 bg-cyan-300/10 p-4 text-sm text-cyan-50">
          <p class="font-medium">知识库管理流程</p>
          <p class="mt-2 text-xs leading-6 text-cyan-100/80">创建知识库 → 上传文档 → 识别内容 → 生成问答 → 管理知识资产</p>
        </div>
      </aside>
      <main class="flex-1 overflow-hidden">
        <header class="flex h-20 items-center justify-between border-b border-white/10 bg-slate-950/30 px-8 backdrop-blur-xl">
          <div>
            <p class="text-xs uppercase tracking-[0.35em] text-cyan-200/70">Knowledge Ops</p>
            <h2 class="mt-1 text-2xl font-semibold">{{ currentTitle }}</h2>
          </div>
          <div class="rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm text-slate-300">本地规则生成 · Go + Vue</div>
        </header>
        <section class="h-[calc(100vh-5rem)] overflow-auto p-8">
          <RouterView />
        </section>
      </main>
    </div>
    <ConfirmDialog />
    <Toast />
  </div>
</template>
