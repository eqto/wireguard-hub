import { writable } from 'svelte/store'

export interface TerminalEntry {
  kind: 'command' | 'output' | 'done'
  command?: string
  line?: string
  error?: string
  timestamp: number
}

export const terminalEntries = writable<Record<string, TerminalEntry[]>>({})
export const terminalExpanded = writable<boolean>(true)

export function addEntry(serverId: string, entry: TerminalEntry) {
  terminalEntries.update((all) => {
    const list = all[serverId] || []
    return {
      ...all,
      [serverId]: [...list, entry],
    }
  })
}

export function clearServer(serverId: string) {
  terminalEntries.update((all) => {
    const next = { ...all }
    delete next[serverId]
    return next
  })
}

export function toggleTerminal() {
  terminalExpanded.update((v) => !v)
}
