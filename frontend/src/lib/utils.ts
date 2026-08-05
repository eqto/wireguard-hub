export function unwrapResponse(result: any): any {
  return result?.data ?? result
}

export function statusColor(status: string): string {
  switch (status) {
    case 'connected':
      return 'var(--success)'
    case 'offline':
      return 'var(--danger)'
    default:
      return 'var(--text-muted)'
  }
}

export function truncateKey(key: string, len: number = 16): string {
  if (key.length <= len) return key
  return key.slice(0, len) + '...'
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

export function formatRelativeTime(date: Date | string): string {
  if (!date) return 'Never'
  const d = typeof date === 'string' ? new Date(date) : date
  if (d.getTime() === 0) return 'Never'
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 0) return 'Just now'
  const seconds = Math.floor(diff / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}
