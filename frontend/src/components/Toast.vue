<script setup lang="ts">
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-vue-next'
import { toastState, hideToast } from '@/composables/useToast'
</script>

<template>
  <Teleport to="body">
    <div
      v-if="toastState.visible"
      class="fixed top-6 right-6 z-[200] flex items-center gap-3 rounded-2xl border px-5 py-4 shadow-2xl transition-all"
      :class="{
        'bg-emerald-400/15 border-emerald-400/30 text-emerald-100': toastState.type === 'success',
        'bg-rose-400/15 border-rose-400/30 text-rose-100': toastState.type === 'error',
        'bg-amber-400/15 border-amber-400/30 text-amber-100': toastState.type === 'warning',
        'bg-cyan-400/15 border-cyan-400/30 text-cyan-100': toastState.type === 'info',
      }"
    >
      <div
        class="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl"
        :class="{
          'bg-emerald-400/20': toastState.type === 'success',
          'bg-rose-400/20': toastState.type === 'error',
          'bg-amber-400/20': toastState.type === 'warning',
          'bg-cyan-400/20': toastState.type === 'info',
        }"
      >
        <CheckCircle v-if="toastState.type === 'success'" class="h-4 w-4" />
        <XCircle v-if="toastState.type === 'error'" class="h-4 w-4" />
        <AlertTriangle v-if="toastState.type === 'warning'" class="h-4 w-4" />
        <Info v-if="toastState.type === 'info'" class="h-4 w-4" />
      </div>
      <p class="text-sm font-medium">{{ toastState.message }}</p>
      <button
        class="ml-2 rounded-lg p-1 transition hover:bg-white/10"
        @click="hideToast"
      >
        <X class="h-4 w-4" />
      </button>
    </div>
  </Teleport>
</template>