<script lang="ts">
  import { onMount } from "svelte";
  import Sidebar from "./lib/components/Sidebar.svelte";
  import ServerGrid from "./lib/components/ServerGrid.svelte";
  import ServerDashboard from "./lib/components/ServerDashboard.svelte";
  import AddServerModal from "./lib/components/AddServerModal.svelte";
  import AddPeerModal from "./lib/components/AddPeerModal.svelte";
  import InterfaceModal from "./lib/components/InterfaceModal.svelte";
  import ConfigViewer from "./lib/components/ConfigViewer.svelte";
  import Terminal from "./lib/components/Terminal.svelte";
  import ThemeToggle from "./lib/components/ThemeToggle.svelte";
  import LocalSetupModal from "./lib/components/LocalSetupModal.svelte";
  import Toaster from "./lib/components/Toaster.svelte";
  import {
    initTheme,
    servers,
    selectedServerId,
    loading,
    error,
  } from "./lib/stores/servers";
  import { showToast } from "./lib/stores/toast";
  import { unwrapResponse } from "./lib/utils";
  import * as ServerService from "../bindings/wireguardhub/internal/server/service.js";
  import * as WireguardService from "../bindings/wireguardhub/internal/wireguard/service.js";

  let showAddServer = $state(false);
  let editingServer = $state<any>(null);
  let showAddPeer = $state(false);
  let showInterfaceModal = $state(false);
  let refreshTrigger = $state(0);
  let showConfigViewer = $state(false);
  let configData = $state({ name: "", content: "" });
  let peerInterface = $state("");
  let peerIsClient = $state(false);
  let showLocalSetup = $state(false);
  let connecting = $state<string | null>(null);

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
    if (id === "local") {
      try {
        const result = await ServerService.GetLocalConfig();
        const cfg = unwrapResponse(result);
        if (!cfg?.configured) {
          showLocalSetup = true;
          return;
        }
      } catch {
        showLocalSetup = true;
        return;
      }
    }
    connecting = id;
    try {
      const result = await WireguardService.GetStatus(id);
      unwrapResponse(result);
      servers.update((list) =>
        list.map((s) =>
          s.id === id ? { ...s, status: "connected" as const } : s,
        ),
      );
      selectedServerId.set(id);
    } catch (e: any) {
      const msg = e?.message || String(e);
      servers.update((list) =>
        list.map((s) =>
          s.id === id ? { ...s, status: "offline" as const } : s,
        ),
      );
      showToast(`Connection failed: ${msg}`, "error");
    } finally {
      connecting = null;
    }
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

  function handleAddPeer(iface: string, isClient: boolean = false) {
    peerInterface = iface;
    peerIsClient = isClient;
    showAddPeer = true;
  }

  function handleCreateInterface() {
    showInterfaceModal = true;
  }

  function handleInterfaceCreated() {
    refreshTrigger++;
  }

  function handleViewConfig(name: string, content: string) {
    configData = { name, content };
    showConfigViewer = true;
  }

  function handleConfigureLocal() {
    showLocalSetup = true;
  }

  let selected = $derived($selectedServerId);

  let viewMode = $state<"grid" | "sidebar">(
    (localStorage.getItem("wg-admin-view-mode") as "grid" | "sidebar") ||
      "grid",
  );

  function toggleViewMode() {
    viewMode = viewMode === "grid" ? "sidebar" : "grid";
    localStorage.setItem("wg-admin-view-mode", viewMode);
    selectedServerId.set(null);
  }
</script>

<div class="app-layout">
  <div class="app-top">
    {#if viewMode === "sidebar"}
      <Sidebar
        onAddServer={handleAddServer}
        onSelect={handleSelectServer}
        onEditServer={handleEditServer}
        onDeleteServer={handleDeleteServer}
        onToggleView={toggleViewMode}
        onConfigureLocal={handleConfigureLocal}
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
            {refreshTrigger}
          />
        {:else}
          <ServerGrid
            onSelect={handleSelectServer}
            onAddServer={handleAddServer}
            onEditServer={handleEditServer}
            onDeleteServer={handleDeleteServer}
            onToggleView={toggleViewMode}
            onConfigureLocal={handleConfigureLocal}
            {connecting}
          />
        {/if}
      {:else if selected}
        <ServerDashboard
          serverId={selected}
          onRefresh={handleRefreshStatus}
          onAddPeer={handleAddPeer}
          onCreateInterface={handleCreateInterface}
          onViewConfig={handleViewConfig}
          onEditServer={handleEditServer}
          onDeleteServer={handleDeleteServer}
          onBack={() => selectedServerId.set(null)}
          {refreshTrigger}
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
    </main>
  </div>
  <Terminal />
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
    isClientInterface={peerIsClient}
    onClose={() => (showAddPeer = false)}
    onAdded={() => refreshTrigger++}
  />
{/if}

{#if showInterfaceModal}
  <InterfaceModal
    serverId={selected}
    onClose={() => (showInterfaceModal = false)}
    onCreated={handleInterfaceCreated}
  />
{/if}

{#if showConfigViewer}
  <ConfigViewer
    name={configData.name}
    content={configData.content}
    onClose={() => (showConfigViewer = false)}
  />
{/if}

{#if showLocalSetup}
  <LocalSetupModal
    onSave={() => {}}
    onConfigured={() => {
      showLocalSetup = false;
      selectedServerId.set("local");
    }}
    onClose={() => (showLocalSetup = false)}
  />
{/if}

<ThemeToggle />
<Toaster />

<style lang="scss">
  .app-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
    width: 100vw;
    overflow: hidden;
    background-color: var(--bg-primary);
  }

  .app-top {
    display: flex;
    flex: 1;
    overflow: hidden;
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
