<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { selectedServerId, servers } from "../stores/servers";
  import {
    terminalEntries,
    terminalExpanded,
    addEntry,
    clearServer,
    toggleTerminal,
    type TerminalEntry,
  } from "../stores/terminal";
  import ChevronDown from "@lucide/svelte/icons/chevron-down";
  import ChevronUp from "@lucide/svelte/icons/chevron-up";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import TerminalIcon from "@lucide/svelte/icons/terminal";

  let terminalBody: HTMLDivElement | null = null;

  let entries = $derived($terminalEntries);
  let expanded = $derived($terminalExpanded);
  let selected = $derived($selectedServerId);
  let serverInfo = $derived($servers.find((s) => s.id === selected));
  let currentEntries = $derived(selected ? entries[selected] || [] : []);

  let offTerminal: (() => void) | null = null;

  onMount(() => {
    console.log("[Terminal] registering ssh-terminal event listener");
    offTerminal = Events.On("ssh-terminal", (event: any) => {
      console.log("[Terminal] received ssh-terminal event:", JSON.stringify(event));
      const data = event.data;
      if (!data || !data.serverId) {
        console.log("[Terminal] skipping event: missing data or serverId", data);
        return;
      }
      const entry: TerminalEntry = {
        kind: data.kind,
        command: data.command,
        line: data.line,
        error: data.error,
        timestamp: Date.now(),
      };
      addEntry(data.serverId, entry);
    });
  });

  onDestroy(() => {
    if (offTerminal) offTerminal();
  });

  $effect(() => {
    if (expanded && terminalBody) {
      terminalBody.scrollTop = terminalBody.scrollHeight;
    }
  });

  function handleClear() {
    if (selected) {
      clearServer(selected);
    }
  }
</script>

<div class="terminal-panel" class:collapsed={!expanded}>
  <div class="terminal-header">
    <button class="terminal-toggle" on:click={toggleTerminal} title={expanded ? "Collapse" : "Expand"}>
      {#if expanded}
        <ChevronDown class="icon-sm" style="color: var(--text-secondary);" />
      {:else}
        <ChevronUp class="icon-sm" style="color: var(--text-secondary);" />
      {/if}
      <TerminalIcon class="icon-sm" style="color: var(--text-secondary);" />
      <span class="terminal-label" style="color: var(--text-secondary);">
        Terminal{#if serverInfo} — {serverInfo.name}{/if}
      </span>
      {#if currentEntries.length > 0}
        <span class="terminal-count" style="color: var(--text-muted);">
          ({currentEntries.length})
        </span>
      {/if}
    </button>
    {#if expanded && selected}
      <button class="terminal-clear-btn" on:click={handleClear} title="Clear terminal">
        <Trash2 class="icon-sm" style="color: var(--text-muted);" />
      </button>
    {/if}
  </div>

  {#if expanded}
    <div class="terminal-body" bind:this={terminalBody}>
      {#if !selected}
        <div class="terminal-empty" style="color: var(--text-muted);">
          Select a server to view its terminal output.
        </div>
      {:else if currentEntries.length === 0}
        <div class="terminal-empty" style="color: var(--text-muted);">
          No commands executed yet.
        </div>
      {:else}
        {#each currentEntries as entry (entry.timestamp)}
          {#if entry.kind === "command"}
            <div class="terminal-line terminal-command">
              <span class="terminal-prompt">$</span>
              <span>{entry.command}</span>
            </div>
          {:else if entry.kind === "output"}
            <div class="terminal-line terminal-output">{entry.line}</div>
          {:else if entry.kind === "done"}
            {#if entry.error}
              <div class="terminal-line terminal-error">Error: {entry.error}</div>
            {/if}
          {/if}
        {/each}
      {/if}
    </div>
  {/if}
</div>

<style lang="scss">
  .terminal-panel {
    display: flex;
    flex-direction: column;
    border-top: 1px solid var(--border);
    background-color: #0c0c0c;
    height: 250px;
    min-height: 250px;
    flex-shrink: 0;

    &.collapsed {
      height: 34px;
      min-height: 34px;
    }
  }

  .terminal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 12px;
    background-color: var(--bg-tertiary);
    border-bottom: 1px solid var(--border);
    height: 34px;
    flex-shrink: 0;
  }

  .terminal-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    border: none;
    background: transparent;
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
  }

  .terminal-label {
    font-size: 13px;
    font-weight: 500;
  }

  .terminal-count {
    font-size: 12px;
  }

  .terminal-clear-btn {
    display: flex;
    align-items: center;
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;

    &:hover {
      background-color: var(--bg-secondary);
    }
  }

  .terminal-body {
    flex: 1;
    overflow-y: auto;
    padding: 8px 12px;
    font-family: "SF Mono", "Monaco", "Inconsolata", "Fira Code", monospace;
    font-size: 13px;
    line-height: 1.5;
  }

  .terminal-empty {
    padding: 8px 0;
    font-size: 13px;
  }

  .terminal-line {
    white-space: pre-wrap;
    word-break: break-all;
  }

  .terminal-command {
    color: #7dd3fc;
    display: flex;
    gap: 6px;
    margin-top: 4px;

    &:first-child {
      margin-top: 0;
    }
  }

  .terminal-prompt {
    color: #22c55e;
    flex-shrink: 0;
  }

  .terminal-output {
    color: #e0e0e0;
  }

  .terminal-error {
    color: #ef4444;
  }
</style>
