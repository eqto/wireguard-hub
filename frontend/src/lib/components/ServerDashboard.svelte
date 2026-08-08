<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { Events } from "@wailsio/runtime";
  import * as WireguardService from "../../../bindings/wireguardhub/internal/wireguard/service.js";
  import { servers, loading, error } from "../stores/servers";
  import { formatBytes, unwrapResponse } from "../utils";
  import StatusBadge from "./StatusBadge.svelte";
  import PeerTable from "./PeerTable.svelte";
  import EditPeerModal from "./EditPeerModal.svelte";
  import ConfirmDialog from "./ConfirmDialog.svelte";
  import RefreshCw from "@lucide/svelte/icons/refresh-cw";
  import ArrowLeft from "@lucide/svelte/icons/arrow-left";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Trash2 from "@lucide/svelte/icons/trash-2";
  import Plus from "@lucide/svelte/icons/plus";
  import FileText from "@lucide/svelte/icons/file-text";
  import Sync from "@lucide/svelte/icons/refresh-ccw";
  import Loader2 from "@lucide/svelte/icons/loader-2";
  import Download from "@lucide/svelte/icons/download";
  import Square from "@lucide/svelte/icons/square";
  import Power from "@lucide/svelte/icons/power";

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
    onAddPeer: (iface: string, isClient: boolean) => void;
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
  let editingPeerIsClient = $state(false);
  let deletingInterface = $state<string | null>(null);
  let wgNotInstalled = $state(false);
  let installing = $state(false);
  let installDone = $state(false);
  let installSuccess = $state(false);
  let installCancelled = $state(false);

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

  function handleDeleteInterface(iface: string) {
    deletingInterface = iface;
  }

  async function confirmDeleteInterface() {
    const iface = deletingInterface;
    deletingInterface = null;
    if (!iface) return;
    try {
      await WireguardService.DeleteInterface(serverId, iface);
      await loadStatus();
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  async function handleBringUp(iface: string) {
    try {
      await WireguardService.BringUpInterface(serverId, iface);
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

  function handleEditPeer(iface: string, peer: any, isClient = false) {
    editingPeerIface = iface;
    editingPeer = peer;
    editingPeerIsClient = isClient;
  }

  async function handleInstallWG() {
    installing = true;
    installDone = false;
    installSuccess = false;
    installCancelled = false;

    offDone = Events.On("wg-install-done", (event: any) => {
      installing = false;
      installDone = true;
      const data = event.data;
      if (data?.success) {
        installSuccess = true;
        setTimeout(() => {
          loadStatus();
        }, 1500);
      }
    });

    try {
      await WireguardService.InstallWireGuard(serverId);
    } catch (e: any) {
      if (!installDone) {
        installing = false;
        installDone = true;
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

  let interfaces = $derived(status?.interfaces || []);
</script>

<div class="dashboard">
  {#if serverInfo}
    <div class="dashboard-header">
      <div class="dashboard-title-row">
        {#if onBack}
          <button
            onclick={onBack}
            class="btn-icon back-btn"
            title="Back to servers"
          >
            <ArrowLeft class="icon" style="color: var(--text-secondary);" />
          </button>
        {/if}
        <div>
          <h1 class="dashboard-title" style="color: var(--text-primary);">
            {serverInfo.name}
          </h1>
          <p class="dashboard-subtitle" style="color: var(--text-muted);">
            {#if serverInfo.isLocal}
              This machine
            {:else}
              {serverInfo.username}@{serverInfo.host}:{serverInfo.port}
            {/if}
          </p>
        </div>
        <StatusBadge status={serverInfo.status} />
      </div>
      <div class="dashboard-actions">
        {#if status}
          <div class="server-endpoint">
            {#if status?.os}
              <span
                class="server-endpoint-line"
                style="color: var(--text-secondary);"
              >
                {status.os}
              </span>
            {/if}
            {#if status?.hostname}
              <span
                class="server-endpoint-line"
                style="color: var(--text-secondary);"
              >
                {status.hostname}
              </span>
            {/if}
            {#if status?.serverIP}
              <span
                class="server-endpoint-line"
                style="color: var(--text-muted);"
              >
                {status.serverIP}
              </span>
            {/if}
          </div>
        {/if}
        <button
          onclick={loadStatus}
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
        {#if !serverInfo.isLocal}
          <button
            onclick={() => onEditServer(serverInfo)}
            class="btn-icon"
            title="Edit server"
          >
            <Pencil class="icon" style="color: var(--text-secondary);" />
          </button>
          <button
            onclick={() => onDeleteServer(serverInfo.id)}
            class="btn-icon"
            title="Delete server"
          >
            <Trash2 class="icon" style="color: var(--danger);" />
          </button>
        {/if}
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
    <div class="dashboard-empty">
      {#if installing}
        <Loader2
          class="icon-lg spin"
          style="color: var(--accent); width: 32px; height: 32px;"
        />
        <p class="dashboard-empty-text" style="color: var(--text-muted);">
          Installing WireGuard... Check terminal for output.
        </p>
        <button onclick={handleCancelInstall} class="btn btn-secondary">
          <Square class="icon-sm" />
          Cancel
        </button>
      {:else if installDone && installSuccess}
        <Loader2
          class="icon-lg spin"
          style="color: var(--accent); width: 32px; height: 32px;"
        />
        <p class="dashboard-empty-text" style="color: var(--text-muted);">
          WireGuard installed successfully. Refreshing...
        </p>
      {:else if installDone}
        <p class="dashboard-empty-text" style="color: var(--text-muted);">
          Installation failed or cancelled. Check terminal for details.
        </p>
        <button onclick={handleInstallWG} class="btn btn-primary">
          <Download class="icon" />
          Retry Install
        </button>
      {:else}
        <p class="dashboard-empty-text" style="color: var(--text-muted);">
          WireGuard is not installed on this server
        </p>
        <button onclick={handleInstallWG} class="btn btn-primary">
          <Download class="icon" />
          Install WireGuard
        </button>
      {/if}
    </div>
  {:else if interfaces.length === 0}
    <div class="dashboard-empty">
      <p class="dashboard-empty-text" style="color: var(--text-muted);">
        No WireGuard interfaces found
      </p>
      <button onclick={onCreateInterface} class="btn btn-primary">
        <Plus class="icon" />
        Create Interface
      </button>
    </div>
  {:else}
    <div class="dashboard-toolbar">
      <button onclick={onCreateInterface} class="btn btn-primary">
        <Plus class="icon" />
        Create Interface
      </button>
    </div>

    {#each interfaces as iface (iface.name)}
      <div
        class="interface-card"
        style="background-color: var(--bg-secondary); border: 1px solid var(--border);"
      >
        {#if iface.listenPort > 0}
          <!-- Server interface card -->
          <div class="interface-header">
            <div class="interface-title-row">
              <h2 class="interface-name" style="color: var(--text-primary);">
                {iface.name}
              </h2>
              {#if !iface.online}
                <span
                  class="interface-port"
                  style="background-color: rgba(220,38,38,0.1); color: var(--danger);"
                >
                  Offline
                </span>
              {:else}
                <span
                  class="interface-port"
                  style="background-color: var(--bg-tertiary); color: var(--text-muted);"
                >
                  Port {iface.listenPort}
                </span>
              {/if}
            </div>
            <div class="interface-actions">
              {#if !iface.online}
                <button
                  onclick={() => handleBringUp(iface.name)}
                  class="btn btn-primary btn-small"
                >
                  <Power class="icon-sm" />
                  Bring Up
                </button>
              {:else}
                <button
                  onclick={() => onAddPeer(iface.name, false)}
                  class="btn btn-primary btn-small"
                >
                  <Plus class="icon-sm" />
                  Add Peer
                </button>
              {/if}
              {#if iface.online}
                <span
                  class="rx-tx-stat"
                  title="Receive / Transfer"
                  style="color: var(--text-muted);"
                >
                  {formatBytes(iface.rxBytes)} / {formatBytes(iface.txBytes)}
                </span>
              {/if}
              <button
                onclick={() => handleViewConfig(iface.name)}
                class="btn-icon btn-icon-small"
                title="View Config"
              >
                <FileText class="icon" style="color: var(--text-secondary);" />
              </button>
              {#if iface.online}
                <button
                  onclick={() => handleSyncConfig(iface.name)}
                  class="btn-icon btn-icon-small"
                  title="Sync Config"
                >
                  <Sync class="icon" style="color: var(--text-secondary);" />
                </button>
              {/if}
              <button
                onclick={() => handleDeleteInterface(iface.name)}
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
                class="stat-value-mono stat-value-full"
                style="color: var(--text-secondary);"
                title={iface.publicKey}
              >
                {#if iface.publicKey}
                  {iface.publicKey}
                {:else}
                  (not running)
                {/if}
              </div>
            </div>
          </div>

          {#if !iface.online}
            {#if iface.peers && iface.peers.length > 0}
              <div class="server-peer-block">
                {#each iface.peers as peer (peer.publicKey)}
                  <div class="server-peer-row">
                    <div class="server-peer-info">
                      <div
                        class="server-peer-name"
                        style="color: var(--text-primary);"
                      >
                        {peer.name || peer.publicKey}
                      </div>
                      <div
                        class="server-peer-detail"
                        style="color: var(--text-muted);"
                      >
                        {peer.endpoint || "No endpoint"}
                      </div>
                      <div
                        class="server-peer-detail"
                        style="color: var(--text-muted);"
                      >
                        {(peer.allowedIPs || []).join(", ")}
                      </div>
                    </div>
                  </div>
                {/each}
              </div>
            {:else}
              <div class="no-server-peer" style="color: var(--text-muted);">
                No peers configured. Bring up the interface to start using it.
              </div>
            {/if}
          {:else}
            <PeerTable
              peers={iface.peers || []}
              onRemove={(pubKey) => handleRemovePeer(iface.name, pubKey)}
              onEdit={(peer) => handleEditPeer(iface.name, peer)}
            />
          {/if}
        {:else}
          <!-- Client interface card -->
          <div class="interface-header">
            <div class="interface-title-row">
              <h2 class="interface-name" style="color: var(--text-primary);">
                {iface.name}
              </h2>
              {#if !iface.online}
                <span
                  class="interface-port"
                  style="background-color: rgba(220,38,38,0.1); color: var(--danger);"
                >
                  Offline
                </span>
              {:else}
                <span
                  class="interface-port"
                  style="background-color: var(--bg-tertiary); color: var(--text-muted);"
                >
                  Client Mode
                </span>
              {/if}
            </div>
            <div class="interface-actions">
              {#if !iface.online}
                <button
                  onclick={() => handleBringUp(iface.name)}
                  class="btn btn-primary btn-small"
                >
                  <Power class="icon-sm" />
                  Bring Up
                </button>
              {/if}
              {#if iface.online}
                <span
                  class="rx-tx-stat"
                  title="Receive / Transfer"
                  style="color: var(--text-muted);"
                >
                  {formatBytes(iface.rxBytes)} / {formatBytes(iface.txBytes)}
                </span>
              {/if}
              <button
                onclick={() => handleViewConfig(iface.name)}
                class="btn-icon btn-icon-small"
                title="View Config"
              >
                <FileText class="icon" style="color: var(--text-secondary);" />
              </button>
              {#if iface.online}
                <button
                  onclick={() => handleSyncConfig(iface.name)}
                  class="btn-icon btn-icon-small"
                  title="Sync Config"
                >
                  <Sync class="icon" style="color: var(--text-secondary);" />
                </button>
              {/if}
              <button
                onclick={() => handleDeleteInterface(iface.name)}
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
                class="stat-value-mono stat-value-full"
                style="color: var(--text-secondary);"
                title={iface.publicKey}
              >
                {#if iface.publicKey}
                  {iface.publicKey}
                {:else}
                  (not running)
                {/if}
              </div>
            </div>
          </div>

          <!-- Server peer block: always shown, even when empty -->
          {#if iface.peers && iface.peers.length > 0}
            <div class="server-peer-block">
              {#each iface.peers as peer (peer.publicKey)}
                <div class="server-peer-row">
                  <div class="server-peer-info">
                    <div class="server-peer-field">
                      <span
                        class="server-peer-label"
                        style="color: var(--text-muted);">Server Address:</span
                      >
                      <span style="color: var(--text-primary);"
                        >{peer.endpoint || "Not configured"}</span
                      >
                    </div>
                    <div class="server-peer-field">
                      <span
                        class="server-peer-label"
                        style="color: var(--text-muted);">Allowed IPs:</span
                      >
                      <span style="color: var(--text-primary);"
                        >{(peer.allowedIPs || []).join(", ") || "None"}</span
                      >
                    </div>
                    {#if peer.name}
                      <div class="server-peer-field">
                        <span
                          class="server-peer-label"
                          style="color: var(--text-muted);">Name:</span
                        >
                        <span style="color: var(--text-primary);"
                          >{peer.name}</span
                        >
                      </div>
                    {/if}
                    <div class="server-peer-field">
                      <span
                        class="server-peer-label"
                        style="color: var(--text-muted);">Public Key:</span
                      >
                      <span
                        class="stat-value-mono stat-value-full"
                        style="color: var(--text-secondary);"
                        >{peer.publicKey}</span
                      >
                    </div>
                  </div>
                  <div class="server-peer-actions">
                    <button
                      onclick={() => handleEditPeer(iface.name, peer, true)}
                      class="btn-icon btn-icon-small"
                      title="Edit"
                    >
                      <Pencil
                        class="icon"
                        style="color: var(--text-secondary);"
                      />
                    </button>
                    <button
                      onclick={() =>
                        handleRemovePeer(iface.name, peer.publicKey)}
                      class="btn-icon btn-icon-small"
                      title="Remove"
                    >
                      <Trash2 class="icon" style="color: var(--danger);" />
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {:else}
            <div class="server-peer-block">
              <div class="server-peer-row">
                <div class="server-peer-info">
                  <div class="no-server-peer" style="color: var(--text-muted);">
                    No server peer configured.
                  </div>
                </div>
                <div class="server-peer-actions">
                  <button
                    onclick={() =>
                      handleEditPeer(
                        iface.name,
                        { publicKey: "", endpoint: "", allowedIPs: [] },
                        true,
                      )}
                    class="btn-icon btn-icon-small"
                    title="Configure"
                  >
                    <Pencil
                      class="icon"
                      style="color: var(--text-secondary);"
                    />
                  </button>
                </div>
              </div>
            </div>
          {/if}
        {/if}
      </div>
    {/each}
  {/if}
</div>

{#if editingPeer}
  <EditPeerModal
    {serverId}
    interfaceName={editingPeerIface}
    peer={editingPeer}
    isClientInterface={editingPeerIsClient}
    onClose={() => {
      editingPeer = null;
      editingPeerIface = "";
      editingPeerIsClient = false;
    }}
    onSaved={loadStatus}
  />
{/if}

{#if deletingInterface}
  <ConfirmDialog
    title="Delete Interface"
    message={`Delete interface ${deletingInterface}? This will bring it down and remove its config file.`}
    confirmLabel="Delete"
    onConfirm={confirmDeleteInterface}
    onCancel={() => (deletingInterface = null)}
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

      .rx-tx-stat {
        font-size: 12px;
        font-family: "Courier New", monospace;
        white-space: nowrap;
        margin-left: 8px;
        margin-right: 4px;
      }
    }

    .interface-stats {
      display: grid;
      grid-template-columns: 1fr;
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

        &-full {
          word-break: break-all;
          white-space: normal;
          overflow: visible;
          text-overflow: clip;
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

  .server-peer-block {
    margin-top: 8px;
  }

  .server-peer-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px;
    border-radius: 8px;
    background-color: var(--bg-tertiary);
    border: 1px solid var(--border);
  }

  .server-peer-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .server-peer-name {
    font-size: 14px;
    font-weight: 600;
  }

  .server-peer-detail {
    font-size: 12px;
  }

  .server-peer-field {
    font-size: 13px;
    display: flex;
    gap: 6px;
  }

  .server-peer-label {
    font-size: 12px;
    font-weight: 500;
    min-width: 120px;
  }

  .server-peer-actions {
    display: flex;
    gap: 4px;
  }

  .no-server-peer {
    font-size: 13px;
    padding: 16px;
    text-align: center;
  }
</style>
