<script lang="ts">
  import { onMount } from "svelte";
  import Sidebar from "./lib/components/Sidebar.svelte";
  import ServerGrid from "./lib/components/ServerGrid.svelte";
  import ServerDashboard from "./lib/components/ServerDashboard.svelte";
  import AddServerModal from "./lib/components/AddServerModal.svelte";
  import AddPeerModal from "./lib/components/AddPeerModal.svelte";
  import InterfaceModal from "./lib/components/InterfaceModal.svelte";
  import ConfigViewer from "./lib/components/ConfigViewer.svelte";
  import ThemeToggle from "./lib/components/ThemeToggle.svelte";
  import {
    initTheme,
    servers,
    selectedServerId,
    loading,
    error,
  } from "./lib/stores/servers";
  import { unwrapResponse } from "./lib/utils";
  import * as ServerService from "../bindings/wireguardadmin/internal/server/service.js";
  import * as WireguardService from "../bindings/wireguardadmin/internal/wireguard/service.js";

  let showAddServer = $state(false);
  let editingServer = $state<any>(null);
  let showAddPeer = $state(false);
  let showInterfaceModal = $state(false);
  let showConfigViewer = $state(false);
  let configData = $state({ name: "", content: "" });
  let peerInterface = $state("");

  onMount(async () => {
    initTheme();
    await loadServers();
  });

  async function loadServers() {
    loading.set(true);
    error.set(null);
    try {
      const result = await ServerService.GetServers();
      const list = unwrapResponse(result) || [];
      servers.set(
        list.map((s: any) => ({
          ...s,
          status: "untested" as const,
        })),
      );
    } catch (e: any) {
      error.set(e?.message || String(e));
    } finally {
      loading.set(false);
    }
  }

  async function handleSelectServer(id: string) {
    selectedServerId.set(id);
  }

  async function handleRefreshStatus() {
    const id = $selectedServerId;
    if (!id) return;
    loading.set(true);
    try {
      await WireguardService.GetStatus(id);
      servers.update((list) =>
        list.map((s) =>
          s.id === id ? { ...s, status: "connected" as const } : s,
        ),
      );
    } catch (e: any) {
      servers.update((list) =>
        list.map((s) =>
          s.id === id ? { ...s, status: "offline" as const } : s,
        ),
      );
      error.set(e?.message || String(e));
    } finally {
      loading.set(false);
    }
  }

  async function handleSaveServer(serverData: any) {
    try {
      if (editingServer) {
        await ServerService.UpdateServer(serverData);
      } else {
        await ServerService.AddServer(serverData);
      }
      showAddServer = false;
      editingServer = null;
      await loadServers();
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  async function handleDeleteServer(id: string) {
    try {
      await ServerService.DeleteServer(id);
      selectedServerId.set(null);
      await loadServers();
    } catch (e: any) {
      error.set(e?.message || String(e));
    }
  }

  function handleEditServer(server: any) {
    editingServer = server;
    showAddServer = true;
  }

  function handleAddServer() {
    editingServer = null;
    showAddServer = true;
  }

  function handleAddPeer(iface: string) {
    peerInterface = iface;
    showAddPeer = true;
  }

  function handleCreateInterface() {
    showInterfaceModal = true;
  }

  function handleViewConfig(name: string, content: string) {
    configData = { name, content };
    showConfigViewer = true;
  }

  let selected = $derived($selectedServerId);

  let viewMode = $state<'grid' | 'sidebar'>(
    (localStorage.getItem('wg-admin-view-mode') as 'grid' | 'sidebar') || 'grid',
  );

  function toggleViewMode() {
    viewMode = viewMode === 'grid' ? 'sidebar' : 'grid';
    localStorage.setItem('wg-admin-view-mode', viewMode);
    selectedServerId.set(null);
  }
</script>

<div class="app-layout">
  {#if viewMode === "sidebar"}
    <Sidebar
      onAddServer={handleAddServer}
      onSelect={handleSelectServer}
      onEditServer={handleEditServer}
      onDeleteServer={handleDeleteServer}
      onToggleView={toggleViewMode}
    />
  {/if}
  <main class="app-main">
    {#if viewMode === "grid"}
      {#if selected}
        <ServerDashboard
          serverId={selected}
          onRefresh={handleRefreshStatus}
          onAddPeer={handleAddPeer}
          onCreateInterface={handleCreateInterface}
          onViewConfig={handleViewConfig}
          onEditServer={handleEditServer}
          onDeleteServer={handleDeleteServer}
          onBack={() => selectedServerId.set(null)}
        />
      {:else}
        <ServerGrid
          onSelect={handleSelectServer}
          onAddServer={handleAddServer}
          onEditServer={handleEditServer}
          onDeleteServer={handleDeleteServer}
          onToggleView={toggleViewMode}
        />
      {/if}
    {:else}
      {#if selected}
        <ServerDashboard
          serverId={selected}
          onRefresh={handleRefreshStatus}
          onAddPeer={handleAddPeer}
          onCreateInterface={handleCreateInterface}
          onViewConfig={handleViewConfig}
          onEditServer={handleEditServer}
          onDeleteServer={handleDeleteServer}
        />
      {:else}
        <div class="app-empty">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            class="app-empty-icon"
            style="color: var(--text-muted);"
          >
            <path d="M12 2L4 7v10l8 5 8-5V7l-8-5z" />
            <path d="M12 22V12" />
            <path d="M4 7l8 5 8-5" />
          </svg>
          <p class="app-empty-text" style="color: var(--text-muted);">
            Select a server to get started
          </p>
        </div>
      {/if}
    {/if}
  </main>
</div>

{#if showAddServer}
  <AddServerModal
    server={editingServer}
    onSave={handleSaveServer}
    onClose={() => {
      showAddServer = false;
      editingServer = null;
    }}
  />
{/if}

{#if showAddPeer}
  <AddPeerModal
    serverId={selected}
    interfaceName={peerInterface}
    onClose={() => (showAddPeer = false)}
  />
{/if}

{#if showInterfaceModal}
  <InterfaceModal
    serverId={selected}
    onClose={() => (showInterfaceModal = false)}
  />
{/if}

{#if showConfigViewer}
  <ConfigViewer
    name={configData.name}
    content={configData.content}
    onClose={() => (showConfigViewer = false)}
  />
{/if}

<ThemeToggle />

<style lang="scss">
  .app-layout {
    display: flex;
    height: 100vh;
    width: 100vw;
    overflow: hidden;
    background-color: var(--bg-primary);
  }

  .app-main {
    flex: 1;
    overflow-y: auto;
    background-color: var(--bg-primary);
  }

  .app-empty {
    display: flex;
    height: 100%;
    align-items: center;
    justify-content: center;
    flex-direction: column;
    gap: 16px;

    &-icon {
      width: 64px;
      height: 64px;
    }

    &-text {
      font-size: 18px;
    }
  }
</style>
