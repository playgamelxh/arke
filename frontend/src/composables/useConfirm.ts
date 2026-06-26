import { reactive } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

export const confirmState = reactive({
  open: false,
  title: '确认操作',
  message: '',
  confirmText: '确认',
  cancelText: '取消',
  danger: false,
  loading: false,
})

let resolver: ((value: boolean) => void) | null = null

export function confirm(options: ConfirmOptions | string): Promise<boolean> {
  const payload = typeof options === 'string' ? { message: options } : options
  confirmState.title = payload.title ?? (payload.danger ? '确认删除' : '确认操作')
  confirmState.message = payload.message
  confirmState.confirmText = payload.confirmText ?? (payload.danger ? '删除' : '确认')
  confirmState.cancelText = payload.cancelText ?? '取消'
  confirmState.danger = payload.danger ?? false
  confirmState.loading = false
  confirmState.open = true

  return new Promise((resolve) => {
    resolver = resolve
  })
}

export function resolveConfirm(value: boolean) {
  confirmState.open = false
  confirmState.loading = false
  resolver?.(value)
  resolver = null
}

export function setConfirmLoading(loading: boolean) {
  confirmState.loading = loading
}
