import { writable } from 'svelte/store'

export interface Toast {
  id: number
  message: string
  type: 'error' | 'success' | 'info'
}

export const toasts = writable<Toast[]>([])

let nextId = 0

export function showToast(message: string, type: Toast['type'] = 'error', duration = 4000) {
  const id = nextId++
  toasts.update((list) => [...list, { id, message, type }])
  setTimeout(() => {
    dismissToast(id)
  }, duration)
}

export function dismissToast(id: number) {
  toasts.update((list) => list.filter((t) => t.id !== id))
}
