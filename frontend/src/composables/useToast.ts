import { reactive } from 'vue'

type ToastType = 'success' | 'error' | 'warning' | 'info'

interface ToastState {
  visible: boolean
  message: string
  type: ToastType
}

export const toastState = reactive<ToastState>({
  visible: false,
  message: '',
  type: 'info',
})

let timeout: ReturnType<typeof setTimeout> | null = null

export function showToast(message: string, type: ToastType = 'info', duration = 3000) {
  if (timeout) {
    clearTimeout(timeout)
  }
  toastState.visible = true
  toastState.message = message
  toastState.type = type
  timeout = setTimeout(() => {
    toastState.visible = false
  }, duration)
}

export function hideToast() {
  if (timeout) {
    clearTimeout(timeout)
    timeout = null
  }
  toastState.visible = false
}

export function success(message: string, duration = 3000) {
  showToast(message, 'success', duration)
}

export function error(message: string, duration = 4000) {
  showToast(message, 'error', duration)
}

export function warning(message: string, duration = 3500) {
  showToast(message, 'warning', duration)
}

export function info(message: string, duration = 3000) {
  showToast(message, 'info', duration)
}

export const toast = {
  success,
  error,
  warning,
  info,
  show: showToast,
  hide: hideToast,
}