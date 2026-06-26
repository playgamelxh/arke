<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { Save, SlidersHorizontal } from 'lucide-vue-next'
import { api } from '@/api/client'
import type { SettingsMap } from '@/types/domain'

const form = reactive<SettingsMap>({
  maxFileSizeMB: '200',
  allowedTypes: 'pdf,ppt,pptx,xls,xlsx,png,jpg,jpeg,doc,docx,md,txt',
  defaultQACount: '10',
  qaGenerateBatchSize: '10',
  modelEndpoint: '',
  modelName: 'qwen-plus',
  storageMode: 'local',
  localUploadDir: '/app/uploads',
  rustfsEndpoint: 'rustfs:9000',
  rustfsAccessKey: '',
  rustfsSecretKey: '',
  rustfsBucket: 'documents',
  rustfsRegion: 'us-east-1',
  rustfsUseSSL: 'false',
  parseEngine: 'auto',
  parsePDFNativeFallback: 'true',
  mineruBaseURL: '',
  mineruTimeoutSeconds: '300',
  mineruParseMethod: 'auto',
  mineruEffort: 'medium',
  mineruLanguage: 'ch',
  mineruImageAnalysis: 'true',
  mineruTableEnable: 'true',
  mineruFormulaEnable: 'true',
})
const message = ref('')

async function load() {
  const settings = await api.settings()
  Object.assign(form, settings)
}

async function save() {
  await api.saveSettings(form)
  message.value = '问答设置已保存'
}

onMounted(load)
</script>

<template>
  <section class="mx-auto max-w-4xl rounded-[2rem] border border-white/10 bg-slate-900/70 p-8">
    <div class="flex items-start justify-between gap-6">
      <div>
        <h3 class="text-xl font-semibold">问答设置</h3>
        <p class="mt-2 text-sm leading-7 text-slate-400">管理问答生成默认参数、批量生成控制和大模型调用配置。</p>
      </div>
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-300/10 text-cyan-200">
        <SlidersHorizontal class="h-6 w-6" />
      </div>
    </div>
    <div class="mt-8 grid gap-5 md:grid-cols-2">
      <label class="block">
        <span class="text-sm text-slate-400">默认生成问答数量</span>
        <input v-model="form.defaultQACount" type="number" min="1" max="50" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
      </label>
      <label class="block">
        <span class="text-sm text-slate-400">每批最多生成条数</span>
        <input v-model="form.qaGenerateBatchSize" type="number" min="1" max="20" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
        <p class="mt-1 text-xs text-slate-500">调用大模型时每批最多生成的问答数，默认 10，最大 20</p>
      </label>
      <label class="block md:col-span-2">
        <span class="text-sm text-slate-400">模型服务地址</span>
        <input v-model="form.modelEndpoint" placeholder="https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
      </label>
      <label class="block md:col-span-2">
        <span class="text-sm text-slate-400">模型名称</span>
        <input v-model="form.modelName" placeholder="qwen-plus" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
      </label>
    </div>
    <button class="mt-8 inline-flex items-center gap-2 rounded-2xl bg-cyan-300 px-6 py-3 font-medium text-slate-950" @click="save"><Save class="h-5 w-5" />保存问答设置</button>
    <p v-if="message" class="mt-5 rounded-2xl border border-cyan-300/20 bg-cyan-300/10 p-4 text-sm text-cyan-100">{{ message }}</p>
  </section>
</template>
