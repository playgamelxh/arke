<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { CheckCircle2, FileSearch, HardDrive, Save, Server, Settings2, TestTube2 } from 'lucide-vue-next'
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

const activeTab = ref<'basic' | 'storage' | 'rustfs' | 'parse'>('basic')
const message = ref('')
const testingRustFS = ref(false)
const rustfsTestMessage = ref('')
const rustfsTestPassed = ref(false)
const testingParse = ref(false)
const parseTestMessage = ref('')
const parseTestPassed = ref(false)

const tabs = [
  { key: 'basic', label: '基础设置', icon: Settings2, desc: '上传限制与通用参数' },
  { key: 'storage', label: '上传位置', icon: HardDrive, desc: '本地 uploads 或 RustFS' },
  { key: 'rustfs', label: 'RustFS 配置', icon: Server, desc: '对象存储连接检测' },
  { key: 'parse', label: '文档解析', icon: FileSearch, desc: '解析引擎与 MinerU 检测' },
] as const

async function load() {
  const settings = await api.settings()
  Object.assign(form, settings)
}

async function save() {
  const saved = await api.saveSettings(form)
  Object.assign(form, saved)
  message.value = '设置已保存，后续操作会使用最新配置'
}

async function testRustFS() {
  testingRustFS.value = true
  rustfsTestMessage.value = ''
  rustfsTestPassed.value = false
  try {
    const result = await api.testRustFS({
      rustfsEndpoint: form.rustfsEndpoint,
      rustfsAccessKey: form.rustfsAccessKey,
      rustfsSecretKey: form.rustfsSecretKey,
      rustfsBucket: form.rustfsBucket,
      rustfsRegion: form.rustfsRegion,
      rustfsUseSSL: form.rustfsUseSSL,
    })
    rustfsTestPassed.value = true
    rustfsTestMessage.value = result.message || 'RustFS 检测通过'
  } catch (err) {
    rustfsTestPassed.value = false
    rustfsTestMessage.value = err instanceof Error ? err.message : 'RustFS 检测失败'
  } finally {
    testingRustFS.value = false
  }
}

async function testParseSettings() {
  testingParse.value = true
  parseTestMessage.value = ''
  parseTestPassed.value = false
  try {
    const result = await api.testParseSettings({
      parseEngine: form.parseEngine,
      parsePDFNativeFallback: form.parsePDFNativeFallback,
      mineruBaseURL: form.mineruBaseURL,
      mineruTimeoutSeconds: form.mineruTimeoutSeconds,
      mineruParseMethod: form.mineruParseMethod,
      mineruEffort: form.mineruEffort,
      mineruLanguage: form.mineruLanguage,
      mineruImageAnalysis: form.mineruImageAnalysis,
      mineruTableEnable: form.mineruTableEnable,
      mineruFormulaEnable: form.mineruFormulaEnable,
    })
    parseTestPassed.value = true
    parseTestMessage.value = result.message || '文档解析配置检测通过'
  } catch (err) {
    parseTestPassed.value = false
    parseTestMessage.value = err instanceof Error ? err.message : '文档解析配置检测失败'
  } finally {
    testingParse.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="mx-auto max-w-5xl space-y-6">
    <section class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-6">
      <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h3 class="text-xl font-semibold">系统设置</h3>
          <p class="mt-2 text-sm leading-7 text-slate-400">通过选项卡管理上传、存储和文档解析配置，避免单页过长。</p>
        </div>
        <button class="inline-flex items-center justify-center gap-2 rounded-2xl bg-cyan-300 px-6 py-3 font-medium text-slate-950" @click="save"><Save class="h-5 w-5" />保存设置</button>
      </div>

      <div class="mt-6 grid gap-3 md:grid-cols-4">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="rounded-2xl border p-4 text-left transition"
          :class="activeTab === tab.key ? 'border-cyan-300/40 bg-cyan-300/10 text-cyan-50' : 'border-white/10 bg-slate-950/60 text-slate-300 hover:bg-white/5'"
          @click="activeTab = tab.key"
        >
          <component :is="tab.icon" class="mb-3 h-5 w-5" />
          <span class="block text-sm font-medium">{{ tab.label }}</span>
          <span class="mt-1 block text-xs text-slate-500">{{ tab.desc }}</span>
        </button>
      </div>

      <p v-if="message" class="mt-5 rounded-2xl border border-cyan-300/20 bg-cyan-300/10 px-4 py-3 text-sm text-cyan-100">{{ message }}</p>
    </section>

    <section v-if="activeTab === 'basic'" class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-8">
      <h3 class="text-xl font-semibold">基础设置</h3>
      <p class="mt-2 text-sm leading-7 text-slate-400">管理系统级上传限制和通用问答生成参数。</p>
      <div class="mt-8 grid gap-5 md:grid-cols-2">
        <label class="block">
          <span class="text-sm text-slate-400">最大文件大小 MB</span>
          <input v-model="form.maxFileSizeMB" type="number" min="1" max="500" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
        </label>
        <label class="block">
          <span class="text-sm text-slate-400">允许上传类型</span>
          <input v-model="form.allowedTypes" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
        </label>
        <label class="block">
          <span class="text-sm text-slate-400">默认问答数量</span>
          <input v-model="form.defaultQACount" type="number" min="1" max="100" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
        </label>
        <label class="block">
          <span class="text-sm text-slate-400">问答生成批次大小</span>
          <input v-model="form.qaGenerateBatchSize" type="number" min="1" max="100" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
        </label>
        <label class="block md:col-span-2">
          <span class="text-sm text-slate-400">模型接口地址</span>
          <input v-model="form.modelEndpoint" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" />
        </label>
      </div>
    </section>

    <section v-else-if="activeTab === 'storage'" class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-8">
      <div class="flex items-start justify-between gap-6">
        <div>
          <h3 class="text-xl font-semibold">文档上传位置</h3>
          <p class="mt-2 text-sm leading-7 text-slate-400">选择知识库上传文件保存到本地 uploads 目录，或保存到 RustFS 对象存储服务。</p>
        </div>
        <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-cyan-300/10 text-cyan-200">
          <HardDrive class="h-6 w-6" />
        </div>
      </div>

      <div class="mt-7 grid gap-4 md:grid-cols-2">
        <button type="button" class="rounded-3xl border p-5 text-left transition" :class="form.storageMode === 'local' ? 'border-cyan-300/40 bg-cyan-300/10 text-cyan-50' : 'border-white/10 bg-slate-950/60 text-slate-300 hover:bg-white/5'" @click="form.storageMode = 'local'">
          <div class="flex items-center gap-3"><HardDrive class="h-5 w-5" /><span class="font-medium">本地 uploads 目录</span></div>
          <p class="mt-3 text-sm leading-6 text-slate-400">文件保存到服务容器内目录，当前 compose 映射到项目 uploads 目录。</p>
        </button>
        <button type="button" class="rounded-3xl border p-5 text-left transition" :class="form.storageMode === 'rustfs' ? 'border-cyan-300/40 bg-cyan-300/10 text-cyan-50' : 'border-white/10 bg-slate-950/60 text-slate-300 hover:bg-white/5'" @click="form.storageMode = 'rustfs'">
          <div class="flex items-center gap-3"><Server class="h-5 w-5" /><span class="font-medium">RustFS 对象存储</span></div>
          <p class="mt-3 text-sm leading-6 text-slate-400">文件保存到 RustFS/S3 兼容 bucket，适合统一管理和持久化。</p>
        </button>
      </div>

      <div class="mt-7 grid gap-5 md:grid-cols-2">
        <label class="block">
          <span class="text-sm text-slate-400">本地上传目录</span>
          <input v-model="form.localUploadDir" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" placeholder="/app/uploads" />
          <p class="mt-1 text-xs text-slate-500">Docker 环境通常为 /app/uploads，对应宿主机项目 uploads 目录。</p>
        </label>
      </div>
    </section>

    <section v-else-if="activeTab === 'rustfs'" class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-8">
      <div class="flex items-start justify-between gap-6">
        <div>
          <h3 class="text-xl font-semibold">RustFS 配置</h3>
          <p class="mt-2 text-sm leading-7 text-slate-400">配置 RustFS/S3 连接信息，并在保存前检测连接、写入、读取和删除是否可用。</p>
        </div>
        <button class="inline-flex items-center gap-2 rounded-2xl border border-cyan-300/30 bg-cyan-300/10 px-4 py-2 text-sm text-cyan-100 disabled:opacity-60" :disabled="testingRustFS" @click="testRustFS">
          <TestTube2 class="h-4 w-4" />{{ testingRustFS ? '检测中...' : '检测 RustFS' }}
        </button>
      </div>

      <div class="mt-8 grid gap-5 md:grid-cols-2">
        <label class="block"><span class="text-sm text-slate-400">Endpoint</span><input v-model="form.rustfsEndpoint" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" placeholder="rustfs:9000" /></label>
        <label class="block"><span class="text-sm text-slate-400">Bucket</span><input v-model="form.rustfsBucket" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" placeholder="documents" /></label>
        <label class="block"><span class="text-sm text-slate-400">Access Key</span><input v-model="form.rustfsAccessKey" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" autocomplete="off" /></label>
        <label class="block"><span class="text-sm text-slate-400">Secret Key</span><input v-model="form.rustfsSecretKey" type="password" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" autocomplete="new-password" /></label>
        <label class="block"><span class="text-sm text-slate-400">Region</span><input v-model="form.rustfsRegion" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" placeholder="us-east-1" /></label>
        <label class="block"><span class="text-sm text-slate-400">是否使用 SSL</span><select v-model="form.rustfsUseSSL" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none"><option value="false">否</option><option value="true">是</option></select></label>
      </div>
      <p v-if="rustfsTestMessage" class="mt-5 rounded-2xl border p-4 text-sm" :class="rustfsTestPassed ? 'border-emerald-300/20 bg-emerald-300/10 text-emerald-100' : 'border-rose-300/20 bg-rose-300/10 text-rose-100'">
        <CheckCircle2 v-if="rustfsTestPassed" class="mr-2 inline h-4 w-4" />{{ rustfsTestMessage }}
      </p>
    </section>

    <section v-else class="rounded-[2rem] border border-white/10 bg-slate-900/70 p-8">
      <div class="flex items-start justify-between gap-6">
        <div>
          <h3 class="text-xl font-semibold">文档解析设置</h3>
          <p class="mt-2 text-sm leading-7 text-slate-400">配置文档解析引擎和 MinerU 参数。保存后，后续解析或重新解析文档会使用这些设置。</p>
        </div>
        <button class="inline-flex items-center gap-2 rounded-2xl border border-cyan-300/30 bg-cyan-300/10 px-4 py-2 text-sm text-cyan-100 disabled:opacity-60" :disabled="testingParse" @click="testParseSettings">
          <TestTube2 class="h-4 w-4" />{{ testingParse ? '检测中...' : '检测解析配置' }}
        </button>
      </div>

      <div class="mt-8 grid gap-5 md:grid-cols-2">
        <label class="block">
          <span class="text-sm text-slate-400">解析引擎</span>
          <select v-model="form.parseEngine" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none">
            <option value="auto">自动：优先 MinerU，失败按规则回退</option>
            <option value="mineru">强制 MinerU</option>
            <option value="native">原生解析</option>
          </select>
          <p class="mt-1 text-xs text-slate-500">Office、图片等复杂格式通常需要 MinerU；原生解析主要适合 PDF、文本和表格。</p>
        </label>
        <label class="block">
          <span class="text-sm text-slate-400">PDF 原生回退</span>
          <select v-model="form.parsePDFNativeFallback" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none"><option value="true">开启</option><option value="false">关闭</option></select>
          <p class="mt-1 text-xs text-slate-500">自动模式下，MinerU 解析 PDF 失败时是否回退到原生 PDF 文本提取。</p>
        </label>
        <label class="block md:col-span-2"><span class="text-sm text-slate-400">MinerU 服务地址</span><input v-model="form.mineruBaseURL" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" placeholder="http://mineru:7861" /></label>
        <label class="block"><span class="text-sm text-slate-400">MinerU 超时时间（秒）</span><input v-model="form.mineruTimeoutSeconds" type="number" min="1" max="600" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" /></label>
        <label class="block"><span class="text-sm text-slate-400">解析方法</span><select v-model="form.mineruParseMethod" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none"><option value="auto">auto</option><option value="txt">txt</option><option value="ocr">ocr</option></select></label>
        <label class="block"><span class="text-sm text-slate-400">解析精度</span><select v-model="form.mineruEffort" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none"><option value="low">low</option><option value="medium">medium</option><option value="high">high</option></select></label>
        <label class="block"><span class="text-sm text-slate-400">语言列表</span><input v-model="form.mineruLanguage" class="mt-2 w-full rounded-2xl border border-white/10 bg-slate-950 px-4 py-3 outline-none" placeholder="ch" /></label>
      </div>

      <div class="mt-6 grid gap-3 md:grid-cols-3">
        <label class="flex items-center justify-between rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-slate-300">图片分析<select v-model="form.mineruImageAnalysis" class="rounded-xl border border-white/10 bg-slate-900 px-3 py-2 outline-none"><option value="true">开启</option><option value="false">关闭</option></select></label>
        <label class="flex items-center justify-between rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-slate-300">表格解析<select v-model="form.mineruTableEnable" class="rounded-xl border border-white/10 bg-slate-900 px-3 py-2 outline-none"><option value="true">开启</option><option value="false">关闭</option></select></label>
        <label class="flex items-center justify-between rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-sm text-slate-300">公式解析<select v-model="form.mineruFormulaEnable" class="rounded-xl border border-white/10 bg-slate-900 px-3 py-2 outline-none"><option value="true">开启</option><option value="false">关闭</option></select></label>
      </div>

      <p v-if="parseTestMessage" class="mt-5 rounded-2xl border p-4 text-sm" :class="parseTestPassed ? 'border-emerald-300/20 bg-emerald-300/10 text-emerald-100' : 'border-rose-300/20 bg-rose-300/10 text-rose-100'">
        <CheckCircle2 v-if="parseTestPassed" class="mr-2 inline h-4 w-4" />{{ parseTestMessage }}
      </p>
    </section>
  </div>
</template>
