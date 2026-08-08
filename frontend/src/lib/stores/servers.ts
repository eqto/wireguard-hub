import { writable } from 'svelte/store'

export type ServerStatus = 'untested' | 'connected' | 'offline'

export interface ServerState {
  id: string
  name: string
  host: string
  port: number
  username: string
  authMethod: string
  viaServerId?: string
  isLocal?: boolean
  status: ServerStatus
}

export const servers = writable<ServerState[]>([])
export const selectedServerId = writable<string | null>(null)
export const loading = writable(false)
export const error = writable<string | null>(null)
export const theme = writable<'dark' | 'light'>('dark')

export function initTheme() {
  const saved = localStorage.getItem('wg-admin-theme')
  if (saved === 'light' || saved === 'dark') {
    theme.set(saved as 'dark' | 'light')
  }
  applyTheme(localStorage.getItem('wg-admin-theme') === 'light' ? 'light' : 'dark')
}

export function toggleTheme() {
  theme.update(t => {
    const next = t === 'dark' ? 'light' : 'dark'
    localStorage.setItem('wg-admin-theme', next)
    applyTheme(next)
    return next
  })
}

function applyTheme(t: 'dark' | 'light') {
  if (t === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}
