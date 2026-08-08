<script lang="ts">
  import { servers } from "../stores/servers";
  import { statusColor } from "../utils";
  import Server from "@lucide/svelte/icons/server";
  import Monitor from "@lucide/svelte/icons/monitor";
  import Plus from "@lucide/svelte/icons/plus";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Shield from "@lucide/svelte/icons/shield";
  import PanelLeft from "@lucide/svelte/icons/panel-left";
  import Settings from "@lucide/svelte/icons/settings";

  let {
    onSelect,
    onAddServer,
    onEditServer,
    onDeleteServer,
    onToggleView,
    onConfigureLocal,
  }: {
    onSelect: (id: string) => void;
    onAddServer: () => void;
    onEditServer: (server: any) => void;
    onDeleteServer: (id: string) => void;
    onToggleView: () => void;
    onConfigureLocal: () => void;
  } = $props();

  let serverList = $derived($servers);
</script>

<div class="server-grid-page">
  <div class="server-grid-header">
    <div class="server-grid-title-row">
      <Shield class="server-grid-icon" style="color: var(--accent);" />
      <h1 class="server-grid-title" style="color: var(--text-primary);">
        WireguardHub
      </h1>
      <button
        onclick={onToggleView}
        class="grid-toggle-btn"
        title="Switch to sidebar view"
      >
        <PanelLeft class="icon" style="color: var(--text-secondary);" />
      </button>
    </div>
    <button onclick={onAddServer} class="btn btn-primary">
      <Plus class="icon" />
      Add Server
    </button>
  </div>

  <div class="server-grid">
    {#each serverList as server (server.id)}
      <div
        class="server-card"
        style="background-color: var(--bg-secondary); border: 1px solid var(--border);"
        onclick={() => onSelect(server.id)}
        onkeydown={(e) => e.key === "Enter" && onSelect(server.id)}
        role="button"
        tabindex="0"
      >
        <div class="server-card-top">
          <div
            class="server-card-dot"
            style="background-color: {statusColor(server.status)};"
          ></div>
          {#if server.isLocal}
            <Monitor
              class="server-card-icon"
              style="color: var(--text-muted);"
            />
          {:else}
            <Server
              class="server-card-icon"
              style="color: var(--text-muted);"
            />
          {/if}
          <div class="server-card-actions">
            {#if server.isLocal}
              <button
                class="server-card-action-btn"
                onclick={(e) => {
                  e.stopPropagation();
                  onConfigureLocal();
                }}
                title="Configure"
              >
                <Settings
                  class="icon-sm"
                  style="color: var(--text-secondary);"
                />
              </button>
            {:else}
              <button
                class="server-card-action-btn"
                onclick={(e) => {
                  e.stopPropagation();
                  onEditServer(server);
                }}
                title="Edit"
              >
                <Pencil class="icon-sm" style="color: var(--text-secondary);" />
              </button>
              <button
                class="server-card-action-btn"
                onclick={(e) => {
                  e.stopPropagation();
                  onDeleteServer(server.id);
                }}
                title="Delete"
              >
                <Trash2 class="icon-sm" style="color: var(--danger);" />
              </button>
            {/if}
          </div>
        </div>
        <div class="server-card-name" style="color: var(--text-primary);">
          {server.name}
        </div>
        <div class="server-card-host" style="color: var(--text-muted);">
          {#if server.isLocal}
            This machine
          {:else}
            {server.username}@{server.host}:{server.port}
          {/if}
        </div>
      </div>
    {/each}
  </div>
</div>

<style lang="scss">
  .server-grid-page {
    padding: 24px;
    height: 100%;
    overflow-y: auto;
  }

  .server-grid-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }

  .server-grid-title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .server-grid-icon {
    width: 20px;
    height: 20px;
  }

  .server-grid-title {
    font-size: 20px;
    font-weight: 700;
  }

  .grid-toggle-btn {
    padding: 4px;
    border: none;
    border-radius: 6px;
    background: transparent;
    cursor: pointer;

    &:hover {
      background-color: var(--bg-tertiary);
    }
  }

  .server-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    max-width: 960px;
  }

  .server-card {
    border-radius: 12px;
    padding: 20px;
    cursor: pointer;
    transition:
      border-color 0.15s,
      transform 0.1s;

    &:hover {
      border-color: var(--accent);
      .server-card-actions {
        opacity: 1;
      }
    }

    &:active {
      transform: scale(0.98);
    }
  }

  .server-card-top {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
  }

  .server-card-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .server-card-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  .server-card-actions {
    display: flex;
    gap: 4px;
    margin-left: auto;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .server-card-action-btn {
    padding: 4px;
    border: none;
    border-radius: 4px;
    background: transparent;
    cursor: pointer;

    &:hover {
      background-color: rgba(0, 0, 0, 0.1);
    }
  }

  .server-card-name {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .server-card-host {
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
