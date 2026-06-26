<script setup lang="ts">
import { AlertTriangle } from 'lucide-vue-next'
import { confirmState, resolveConfirm } from '@/composables/useConfirm'
</script>

<template>
  <Teleport to="body">
    <div
      v-if="confirmState.open"
      class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/70 p-4 backdrop-blur-sm"
      @click.self="resolveConfirm(false)"
    >
      <div
        class="w-full max-w-md rounded-[2rem] border border-white/10 bg-slate-900 p-6 shadow-2xl"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="confirmState.danger ? 'confirm-dialog-title-danger' : 'confirm-dialog-title'"
      >
        <div class="flex items-start gap-4">
          <div
            class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl"
            :class="confirmState.danger ? 'bg-rose-400/15 text-rose-100' : 'bg-cyan-300/15 text-cyan-100'"
          >
            <AlertTriangle class="h-6 w-6" />
          </div>
          <div class="min-w-0 flex-1">
            <h3
              :id="confirmState.danger ? 'confirm-dialog-title-danger' : 'confirm-dialog-title'"
              class="text-lg font-semibold text-slate-100"
            >
              {{ confirmState.title }}
            </h3>
            <p class="mt-2 text-sm leading-6 text-slate-400">{{ confirmState.message }}</p>
          </div>
        </div>
        <div class="mt-6 flex gap-3">
          <button
            type="button"
            class="flex-1 rounded-2xl px-5 py-3 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-60"
            :class="confirmState.danger ? 'bg-rose-400 text-slate-950 hover:bg-rose-300' : 'bg-cyan-300 text-slate-950 hover:bg-cyan-200'"
            :disabled="confirmState.loading"
            @click="resolveConfirm(true)"
          >
            {{ confirmState.loading ? '处理中...' : confirmState.confirmText }}
          </button>
          <button
            type="button"
            class="rounded-2xl bg-white/10 px-5 py-3 text-sm hover:bg-white/15 disabled:opacity-60"
            :disabled="confirmState.loading"
            @click="resolveConfirm(false)"
          >
            {{ confirmState.cancelText }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
