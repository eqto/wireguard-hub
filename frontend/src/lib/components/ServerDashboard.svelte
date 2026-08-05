<script lang="ts">
  import { onMount } from "svelte";
  import * as WireguardService from "../../../bindings/wireguardadmin/internal/wireguard/service.js";
  import { servers, loading, error } from "../stores/servers";
  import { formatBytes, unwrapResponse } from "../utils";
  import StatusBadge from "./StatusBadge.svelte";
  import PeerTable from "./PeerTable.svelte";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Plus from "@lucide/svelte/icons/plus";
  import FileText from "@lucide/svelte/icons/file-text";
  import Sync from "@lucide/svelte/icons/refresh-ccw";
  import Loader2 from "@lucide/svelte/icons/loader-2";

  let {
    serverId,
    onRefresh,
    onAddPeer,
    onCreateInterface,
    onViewConfig,
    onEditServer,
    onDeleteServer,
  }: {
    serverId: string;
    onRefresh: () => void;
    onAddPeer: (iface: string) => void;
    onCreateInterface: () => void;
    onViewConfig: (name: string, content: string) => void;
    onEditServer: (server: any) => void;
    onDeleteServer: (id: string) => void;
  } = $props();

  let status = $state<any>(null);
  let isLoading = $state(false);
  let serverInfo = $derived($servers.find((s) => s.id === serverId));

  onMount(() => {
    loadStatus();
  });

  async function loadStatus() {
    isLoading = true;
    error.set(null);
    try {
      const result = await WireguardService.GetStatus(serverId);
      status = unwrapResponse(result);
      servers.update((list) =>
        list.map((s) =>
          s.id === serverId ? { ...s, status: "connected" as const } : s,
        ),
      );
    } catch (e: any) {
      servers.update((list) =>
        list.map((s) =>
          s.id === serverId ? { ...s, status: "offline" as const } : s,
        ),
      );
      error.set(e?.message || String(e));
    } finally {
      isLoading = false;
    }
  }

  async function handleSyncConfig(iface: string) {
    try {
      await WireguardService.SyncConfig(serverId, iface);
      await loadStatus();
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  async function handleViewConfig(iface: string) {
    try {
      const result = await WireguardService.GetInterfaceConfig(serverId, iface);
      const content = unwrapResponse(result) || "";
      onViewConfig(iface, content);
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  async function handleDeleteInterface(iface: string) {
    if (
      !confirm(
        `Delete interface ${iface}? This will bring it down and remove its config file.`,
      )
    )
      return;
    try {
      await WireguardService.DeleteInterface(serverId, iface);
      await loadStatus();
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  async function handleRemovePeer(iface: string, publicKey: string) {
    try {
      await WireguardService.RemovePeer(serverId, iface, publicKey);
      await loadStatus();
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  let interfaces = $derived(status?.interfaces || []);
</script>

<div class="dashboard">
  {#if serverInfo}
    <div class="dashboard-header">
      <div class="dashboard-title-row">
        <div>
          <h1 class="dashboard-title" style="color: var(--text-primary);">
            {serverInfo.name}
          </h1>
          <p class="dashboard-subtitle" style="color: var(--text-muted);">
            {serverInfo.username}@{serverInfo.host}:{serverInfo.port}
          </p>
        </div>
        <StatusBadge status={serverInfo.status} />
      </div>
      <div class="dashboard-actions">
        <button
          on:click={loadStatus}
          disabled={isLoading}
          class="btn btn-secondary"
        >
          {#if isLoading}
            <Loader2 class="icon spin" />
          {:else}
            <RefreshCw class="icon" />
          {/if}
          Refresh
        </button>
        <button
          on:click={() => onEditServer(serverInfo)}
          class="btn-icon"
          title="Edit server"
        >
          <Pencil class="icon" style="color: var(--text-secondary);" />
        </button>
        <button
          on:click={() => onDeleteServer(serverInfo.id)}
          class="btn-icon"
          title="Delete server"
        >
          <Trash2 class="icon" style="color: var(--danger);" />
        </button>
      </div>
    </div>
  {/if}

  {#if isLoading && !status}
    <div class="dashboard-loading">
      <Loader2
        class="icon-lg spin"
        style="color: var(--accent); width: 32px; height: 32px;"
      />
    </div>
  {:else if interfaces.length === 0}
    <div class="dashboard-empty">
      <p class="dashboard-empty-text" style="color: var(--text-muted);">
        No WireGuard interfaces found
      </p>
      <button on:click={onCreateInterface} class="btn btn-primary">
        <Plus class="icon" />
        Create Interface
      </button>
    </div>
  {:else}
    <div class="dashboard-toolbar">
      <button on:click={onCreateInterface} class="btn btn-primary">
        <Plus class="icon" />
        Create Interface
      </button>
    </div>

    {#each interfaces as iface (iface.name)}
      <div
        class="interface-card"
        style="background-color: var(--bg-secondary); border: 1px solid var(--border);"
      >
        <div class="interface-header">
          <div class="interface-title-row">
            <h2 class="interface-name" style="color: var(--text-primary);">
              {iface.name}
            </h2>
            <span
              class="interface-port"
              style="background-color: var(--bg-tertiary); color: var(--text-muted);"
            >
              Port {iface.listenPort}
            </span>
          </div>
          <div class="interface-actions">
            <button
              on:click={() => onAddPeer(iface.name)}
              class="btn btn-primary btn-small"
            >
              <Plus class="icon-sm" />
              Add Peer
            </button>
            <button
              on:click={() => handleViewConfig(iface.name)}
              class="btn-icon btn-icon-small"
              title="View Config"
            >
              <FileText class="icon" style="color: var(--text-secondary);" />
            </button>
            <button
              on:click={() => handleSyncConfig(iface.name)}
              class="btn-icon btn-icon-small"
              title="Sync Config"
            >
              <Sync class="icon" style="color: var(--text-secondary);" />
            </button>
            <button
              on:click={() => handleDeleteInterface(iface.name)}
              class="btn-icon btn-icon-small"
              title="Delete Interface"
            >
              <Trash2 class="icon" style="color: var(--danger);" />
            </button>
          </div>
        </div>

        <div class="interface-stats">
          <div>
            <div class="stat-label" style="color: var(--text-muted);">
              Public Key
            </div>
            <div
              class="stat-value-mono"
              style="color: var(--text-secondary);"
              title={iface.publicKey}
            >
              {iface.publicKey?.slice(0, 20)}...
            </div>
          </div>
          <div>
            <div class="stat-label" style="color: var(--text-muted);">RX</div>
            <div class="stat-value" style="color: var(--text-secondary);">
              {formatBytes(iface.rxBytes)}
            </div>
          </div>
          <div>
            <div class="stat-label" style="color: var(--text-muted);">TX</div>
            <div class="stat-value" style="color: var(--text-secondary);">
              {formatBytes(iface.txBytes)}
            </div>
          </div>
        </div>

        <PeerTable
          peers={iface.peers || []}
          onRemove={(pubKey) => handleRemovePeer(iface.name, pubKey)}
        />
      </div>
    {/each}
  {/if}
</div>

<style lang="scss">
  .dashboard {
    padding: 24px;

    &-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 24px;
    }

    &-title-row {
      display: flex;
      align-items: center;
      gap: 16px;
    }

    &-title {
      font-size: 20px;
      font-weight: 700;
    }

    &-subtitle {
      font-size: 14px;
    }

    &-actions {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    &-loading {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 80px 0;
    }

    &-empty {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 80px 0;
      gap: 16px;

      &-text {
        font-size: 14px;
      }
    }

    &-toolbar {
      display: flex;
      justify-content: flex-end;
      margin-bottom: 16px;
    }
  }

  .interface-card {
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 16px;

    .interface-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 16px;

      .interface-title-row {
        display: flex;
        align-items: center;
        gap: 12px;
      }

      .interface-name {
        font-size: 18px;
        font-weight: 600;
      }

      .interface-port {
        font-size: 12px;
        padding: 2px 8px;
        border-radius: 9999px;
      }

      .interface-actions {
        display: flex;
        align-items: center;
        gap: 4px;
      }
    }

    .interface-stats {
      display: grid;
      grid-template-columns: 1fr 1fr 1fr;
      gap: 16px;
      margin-bottom: 16px;

      .stat-label {
        font-size: 12px;
        margin-bottom: 4px;
      }

      .stat-value {
        font-size: 14px;

        &-mono {
          font-size: 14px;
          font-family: "Courier New", monospace;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }
    }
  }

  .btn-small {
    padding: 6px 12px;
    font-size: 12px;
  }

  .btn-icon-small {
    padding: 6px;
  }
</style>
