<script lang="ts">
  import { servers, selectedServerId, loading } from "../stores/servers";
  import { statusColor } from "../utils";
  import Server from "@lucide/svelte/icons/server";
  import Plus from "@lucide/svelte/icons/plus";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Shield from "@lucide/svelte/icons/shield";

  let {
    onAddServer,
    onSelect,
    onEditServer,
    onDeleteServer,
  }: {
    onAddServer: () => void;
    onSelect: (id: string) => void;
    onEditServer: (server: any) => void;
    onDeleteServer: (id: string) => void;
  } = $props();

  let serverList = $derived($servers);
  let selected = $derived($selectedServerId);
  let isLoading = $derived($loading);

  function jumpName(viaId: string, list: any[]): string {
    if (!viaId) return "";
    const j = list.find((s) => s.id === viaId);
    return j ? j.name : "?";
  }
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <Shield class="sidebar-icon" style="color: var(--accent);" />
    <span class="sidebar-title" style="color: var(--text-primary);"
      >WireGuard Admin</span
    >
  </div>

  <div class="sidebar-list">
    {#if serverList.length === 0}
      <div class="sidebar-empty">
        <Server class="sidebar-empty-icon" style="color: var(--text-muted);" />
        <p class="sidebar-empty-text" style="color: var(--text-muted);">
          No servers yet
        </p>
        <p class="sidebar-empty-sub" style="color: var(--text-muted);">
          Click + to add one
        </p>
      </div>
    {:else}
      {#each serverList as server (server.id)}
        <div
          class="server-item"
          style={selected === server.id
            ? "background-color: var(--bg-tertiary);"
            : ""}
          on:click={() => onSelect(server.id)}
          on:keydown={(e) => e.key === "Enter" && onSelect(server.id)}
          role="button"
          tabindex="0"
        >
          <div
            class="server-status-dot"
            style="background-color: {statusColor(server.status)};"
          ></div>
          <div class="server-info">
            <div class="server-name" style="color: var(--text-primary);">
              {server.name}
            </div>
            <div class="server-host" style="color: var(--text-muted);">
              {server.host}:{server.port}{#if server.viaServerId}
                <span class="server-via">via {jumpName(server.viaServerId, serverList)}</span>{/if}
            </div>
          </div>
          <div class="server-actions">
            <button
              class="server-action-btn"
              on:click={(e) => {
                e.stopPropagation();
                onEditServer(server);
              }}
              title="Edit"
            >
              <Pencil
                class="server-action-icon"
                style="color: var(--text-secondary);"
              />
            </button>
            <button
              class="server-action-btn"
              on:click={(e) => {
                e.stopPropagation();
                onDeleteServer(server.id);
              }}
              title="Delete"
            >
              <Trash2
                class="server-action-icon"
                style="color: var(--danger);"
              />
            </button>
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <div class="sidebar-footer" style="border-color: var(--border);">
    <button class="add-server-btn" on:click={onAddServer}>
      <Plus class="add-server-icon" />
      <span>Add Server</span>
    </button>
  </div>
</aside>

<style lang="scss">
  .sidebar {
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 280px;
    min-width: 280px;
    border-right: 1px solid var(--border);
    background-color: var(--bg-secondary);

    &-header {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 16px;
      border-bottom: 1px solid var(--border);
    }

    &-icon {
      width: 20px;
      height: 20px;
    }

    &-title {
      font-weight: 600;
      font-size: 16px;
    }

    &-list {
      flex: 1;
      overflow-y: auto;
      padding: 8px;
    }

    &-empty {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 48px 16px;
      text-align: center;

      &-icon {
        width: 32px;
        height: 32px;
        margin-bottom: 8px;
      }

      &-text {
        font-size: 14px;
        margin: 0;
      }

      &-sub {
        font-size: 12px;
        margin-top: 4px;
      }
    }

    &-footer {
      padding: 16px;
      border-top: 1px solid var(--border);
    }
  }

  .server-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: 8px;
    cursor: pointer;
    transition: background-color 0.15s;
    margin-bottom: 4px;

    &:hover .server-actions {
      display: flex;
    }

    .server-status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      flex-shrink: 0;
    }

    .server-info {
      flex: 1;
      min-width: 0;
    }

    .server-name {
      font-size: 14px;
      font-weight: 500;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .server-host {
      font-size: 12px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .server-via {
      margin-left: 6px;
      opacity: 0.7;
      font-style: italic;
    }

    .server-actions {
      display: none;
      align-items: center;
      gap: 4px;
    }

    .server-action-btn {
      padding: 4px;
      border: none;
      border-radius: 4px;
      background: transparent;
      cursor: pointer;

      &:hover {
        background-color: rgba(0, 0, 0, 0.1);
      }
    }

    .server-action-icon {
      width: 14px;
      height: 14px;
    }
  }

  .add-server-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    width: 100%;
    padding: 12px 16px;
    border: none;
    border-radius: 8px;
    font-weight: 500;
    font-size: 14px;
    background-color: var(--accent);
    color: white;
    cursor: pointer;
    transition: opacity 0.15s;

    &:hover {
      opacity: 0.9;
    }

    &-icon {
      width: 16px;
      height: 16px;
    }
  }
</style>
