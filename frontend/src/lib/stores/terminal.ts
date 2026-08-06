import { writable } from 'svelte/store'

export interface TerminalEntry {
  id: number
  kind: 'command' | 'output' | 'done'
  command?: string
  line?: string
  error?: string
  timestamp: number
}

export const terminalEntries = writable<Record<string, TerminalEntry[]>>({})
export const terminalExpanded = writable<boolean>(true)

let nextEntryId = 0

export function addEntry(serverId: string, entry: Omit<TerminalEntry, 'id'>) {
  terminalEntries.update((all) => {
    const list = all[serverId] || []
    return {
      ...all,
      [serverId]: [...list, { ...entry, id: nextEntryId++ }],
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
