export function formatDate(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

export function formatSize(value: number) {
  if (!value) return '0 B'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

export function statusText(status: string) {
  const map: Record<string, string> = {
    uploaded: '已上传',
    parsing: '解析中',
    parsed: '已解析',
    failed: '解析失败',
  }
  return map[status] || status
}

export function statusClass(status: string) {
  const map: Record<string, string> = {
    uploaded: 'bg-slate-500/20 text-slate-200 border-slate-400/20',
    parsing: 'bg-amber-400/20 text-amber-100 border-amber-300/30',
    parsed: 'bg-emerald-400/20 text-emerald-100 border-emerald-300/30',
    failed: 'bg-rose-400/20 text-rose-100 border-rose-300/30',
  }
  return map[status] || map.uploaded
}
