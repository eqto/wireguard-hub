<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import * as WireguardService from "../../../bindings/wireguardadmin/internal/wireguard/service.js";
  import { servers, loading, error } from "../stores/servers";
  import { formatBytes, unwrapResponse } from "../utils";
  import StatusBadge from "./StatusBadge.svelte";
  import PeerTable from "./PeerTable.svelte";
  import EditPeerModal from "./EditPeerModal.svelte";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Plus from "@lucide/svelte/icons/plus";
  import FileText from "@lucide/svelte/icons/file-text";
  import Sync from "@lucide/svelte/icons/refresh-ccw";
  import Loader2 from "@lucide/svelte/icons/loader-2";
  import Download from "@lucide/svelte/icons/download";
  import X from "@lucide/svelte/icons/x";
  import Square from "@lucide/svelte/icons/square";

  let {
    serverId,
    onRefresh,
    onAddPeer,
    onCreateInterface,
    onViewConfig,
    onEditServer,
    onDeleteServer,
    onBack,
    refreshTrigger = 0,
  }: {
    serverId: string;
    onRefresh: () => void;
    onAddPeer: (iface: string) => void;
    onCreateInterface: () => void;
    onViewConfig: (name: string, content: string) => void;
    onEditServer: (server: any) => void;
    onDeleteServer: (id: string) => void;
    onBack?: () => void;
    refreshTrigger?: number;
  } = $props();

  let status = $state<any>(null);
  let isLoading = $state(false);
  let serverInfo = $derived($servers.find((s) => s.id === serverId));
  let editingPeer = $state<any>(null);
  let editingPeerIface = $state("");
  let wgNotInstalled = $state(false);
  let installing = $state(false);
  let installOutput = $state<string[]>([]);
  let installDone = $state(false);
  let installCancelled = $state(false);
  let terminalEl: HTMLDivElement | null = null;

  let offOutput: (() => void) | null = null;
  let offDone: (() => void) | null = null;

  onMount(() => {
    loadStatus();
  });

  $effect(() => {
    if (refreshTrigger > 0) {
      loadStatus();
    }
  });

  onDestroy(() => {
    if (offOutput) offOutput();
    if (offDone) offDone();
  });

  async function loadStatus() {
    isLoading = true;
    error.set(null);
    wgNotInstalled = false;
    try {
      const result = await WireguardService.GetStatus(serverId);
      status = unwrapResponse(result);
      wgNotInstalled = status?.wgNotInstalled ?? false;
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

  function handleEditPeer(iface: string, peer: any) {
    editingPeerIface = iface;
    editingPeer = peer;
  }

  async function handleInstallWG() {
    installing = true;
    installDone = false;
    installCancelled = false;
    installOutput = [];

    offOutput = Events.On("wg-install-output", (event: any) => {
      installOutput = [...installOutput, event.data];
      if (terminalEl) {
        terminalEl.scrollTop = terminalEl.scrollHeight;
      }
    });

    offDone = Events.On("wg-install-done", (event: any) => {
      installing = false;
      installDone = true;
      const data = event.data;
      if (data?.success) {
        installOutput = [...installOutput, "", "Installation completed successfully."];
        setTimeout(() => {
          loadStatus();
        }, 1500);
      } else if (installCancelled) {
        installOutput = [...installOutput, "", "Installation cancelled."];
      } else {
        installOutput = [...installOutput, "", `Installation failed: ${data?.error || "unknown error"}`];
      }
    });

    try {
      await WireguardService.InstallWireGuard(serverId);
    } catch (e: any) {
      if (!installDone) {
        installing = false;
        installDone = true;
        installOutput = [...installOutput, "", `Error: ${e?.message || String(e)}`];
      }
    }
  }

  async function handleCancelInstall() {
    installCancelled = true;
    try {
      await WireguardService.CancelInstall();
    } catch (e: any) {
      // ignore
    }
  }

  function handleCloseTerminal() {
    installDone = false;
    installOutput = [];
  }

  let interfaces = $derived(status?.interfaces || []);
</script>

<div class="dashboard">
  {#if serverInfo}
    <div class="dashboard-header">
      <div class="dashboard-title-row">
        {#if onBack}
          <button on:click={onBack} class="btn-icon back-btn" title="Back to servers">
            <ArrowLeft class="icon" style="color: var(--text-secondary);" />
          </button>
        {/if}
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
        {#if status}
          <div class="server-endpoint">
            {#if status?.os}
              <span class="server-endpoint-line" style="color: var(--text-secondary);">
                {status.os}
              </span>
            {/if}
            {#if status?.hostname}
              <span class="server-endpoint-line" style="color: var(--text-secondary);">
                {status.hostname}
              </span>
            {/if}
            {#if status?.serverIP}
              <span class="server-endpoint-line" style="color: var(--text-muted);">
                {status.serverIP}
              </span>
            {/if}
          </div>
        {/if}
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
  {:else if wgNotInstalled}
    {#if installDone || installing}
      <div class="install-terminal-wrapper">
        <div class="install-terminal-header">
          <span class="install-terminal-title" style="color: var(--text-secondary);">
            {#if installing}Installing WireGuard...{:else}Installation output{/if}
          </span>
          {#if installing}
            <button on:click={handleCancelInstall} class="btn btn-secondary btn-sm">
              <Square class="icon-sm" />
              Cancel
            </button>
          {:else}
            <button on:click={handleCloseTerminal} class="btn-icon" title="Close terminal">
              <X class="icon" style="color: var(--text-secondary);" />
            </button>
          {/if}
        </div>
        <div class="install-terminal" bind:this={terminalEl}>
          {#each installOutput as line}
            <div class="terminal-line">{line}</div>
          {/each}
        </div>
      </div>
    {:else}
      <div class="dashboard-empty">
        <p class="dashboard-empty-text" style="color: var(--text-muted);">
          WireGuard is not installed on this server
        </p>
        <button on:click={handleInstallWG} class="btn btn-primary">
          <Download class="icon" />
          Install WireGuard
        </button>
      </div>
    {/if}
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
          onEdit={(peer) => handleEditPeer(iface.name, peer)}
        />
      </div>
    {/each}
  {/if}
</div>

{#if editingPeer}
  <EditPeerModal
    serverId={serverId}
    interfaceName={editingPeerIface}
    peer={editingPeer}
    onClose={() => {
      editingPeer = null;
      editingPeerIface = "";
    }}
    onSaved={loadStatus}
  />
{/if}

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

  .back-btn {
    padding: 6px;
    border: none;
    border-radius: 8px;
    background: transparent;
    cursor: pointer;

    &:hover {
      background-color: var(--bg-tertiary);
    }
  }

  .server-endpoint {
    display: flex;
    flex-direction: column;
    margin-right: auto;
  }

  .server-endpoint-line {
    font-size: 13px;
    line-height: 1.3;
  }

  .install-terminal-wrapper {
    margin: 16px 0;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--border);
  }

  .install-terminal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background-color: var(--bg-tertiary);
    border-bottom: 1px solid var(--border);
  }

  .install-terminal-title {
    font-size: 13px;
    font-weight: 500;
  }

  .install-terminal {
    background-color: #0c0c0c;
    padding: 12px;
    height: 400px;
    overflow-y: auto;
    font-family: "SF Mono", "Monaco", "Inconsolata", "Fira Code", monospace;
    font-size: 13px;
    line-height: 1.5;
  }

  .terminal-line {
    color: #e0e0e0;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .btn-sm {
    padding: 4px 10px;
    font-size: 12px;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .icon-sm {
    width: 12px;
    height: 12px;
  }
</style>
