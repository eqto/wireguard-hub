<script lang="ts">
  import { X, LoaderCircle, Sparkles } from "@lucide/svelte";
  import { onMount } from "svelte";
  import * as WireguardService from "../../../bindings/wireguardhub/internal/wireguard/service.js";
  import { error } from "../stores/servers";
  import { unwrapResponse } from "../utils";

  let {
    serverId,
    onClose,
    onCreated,
  }: {
    serverId: string;
    onClose: () => void;
    onCreated?: () => void;
  } = $props();

  let name = $state("wg0");
  let mode = $state<"server" | "client">("server");
  let listenPort = $state(51820);
  let privateKey = $state("");
  let address = $state("10.0.0.1/24");
  let endpoint = $state("");
  let generating = $state(false);
  let creating = $state(false);
  let enableService = $state(false);

  onMount(() => {
    handleGenerate();
  });

  async function handleGenerate() {
    generating = true;
    try {
      const res = await WireguardService.GenerateKeyPair(serverId);
      const kp = unwrapResponse(res);
      privateKey = kp.privateKey;
    } catch (e: any) {
      error.set(e?.message || String(e));
    } finally {
      generating = false;
    }
  }

  async function handleCreate() {
    creating = true;
    try {
      const req = {
        serverId: serverId,
        name: name,
        listenPort: mode === "server" ? Number(listenPort) : 0,
        privateKey: privateKey,
        address: address,
        endpoint: endpoint,
        allowedIPs: [],
        enableService: enableService,
      };
      await WireguardService.CreateInterface(req);
      if (onCreated) onCreated();
      onClose();
    } catch (e: any) {
      error.set(e?.message || String(e));
    } finally {
      creating = false;
    }
  }
</script>

<div
  class="modal-overlay"
  onclick={onClose}
  onkeydown={(e) => e.key === "Escape" && onClose()}
  role="button"
  tabindex="0"
>
  <div
    class="modal"
    onclick={(e) => e.stopPropagation()}
    role="dialog"
    tabindex="0"
  >
    <div class="modal-header">
      <h2 class="modal-title">Create Interface</h2>
      <button onclick={onClose} class="close-btn">
        <X class="icon-lg" style="color: var(--text-secondary);" />
      </button>
    </div>

    <div class="modal-body">
      <div class="form-group">
        <label class="label">Mode</label>
        <div class="mode-toggle">
          <button
            type="button"
            onclick={() => (mode = "server")}
            class="mode-btn"
            class:active={mode === "server"}
          >
            Server
          </button>
          <button
            type="button"
            onclick={() => (mode = "client")}
            class="mode-btn"
            class:active={mode === "client"}
          >
            Client
          </button>
        </div>
      </div>

      {#if mode === "client"}
        <div class="form-group">
          <label class="label">Server Address</label>
          <input
            bind:value={endpoint}
            type="text"
            placeholder="vpn.example.com:51820"
            class="input"
          />
        </div>
      {/if}

      <div class="form-row">
        <div class="form-group form-grow">
          <label class="label">Interface Name</label>
          <input
            bind:value={name}
            type="text"
            placeholder="wg0"
            class="input"
          />
        </div>
        {#if mode === "server"}
          <div class="form-group form-fixed-128">
            <label class="label">Listen Port</label>
            <input
              bind:value={listenPort}
              type="number"
              placeholder="51820"
              class="input"
            />
          </div>
        {/if}
      </div>

      <div class="form-group">
        <label class="label">Address</label>
        <input
          bind:value={address}
          type="text"
          placeholder="10.0.0.1/24"
          class="input"
        />
      </div>

      <div class="form-group">
        <label class="label">Private Key</label>
        <div class="input-row">
          <input
            bind:value={privateKey}
            type="text"
            placeholder="Auto-generated or paste existing"
            class="input input-mono"
            style="flex: 1;"
          />
          <button
            onclick={handleGenerate}
            disabled={generating}
            class="btn btn-secondary"
            title="Generate keypair on server"
          >
            {#if generating}
              <LoaderCircle class="icon spin" />
            {:else}
              <Sparkles class="icon" />
            {/if}
            Generate
          </button>
        </div>
      </div>

      <div class="form-group">
        <label class="checkbox-row">
          <input
            bind:checked={enableService}
            type="checkbox"
            class="checkbox"
          />
          <span>Start as systemd service (autostart on boot)</span>
        </label>
        <p class="checkbox-hint">
          When enabled, the interface is managed via
          <code>wg-quick@{name || "wg0"}</code> and starts automatically on boot.
          Requires systemd on the server.
        </p>
      </div>
    </div>

    <div class="modal-footer">
      <button
        onclick={onClose}
        class="btn-text"
        style="color: var(--text-secondary);"
      >
        Cancel
      </button>
      <button
        onclick={handleCreate}
        disabled={creating ||
          !name ||
          (mode === "server" && !listenPort) ||
          (mode === "client" && !endpoint)}
        class="btn btn-primary"
      >
        {#if creating}
          <LoaderCircle class="icon spin" />
        {/if}
        Create
      </button>
    </div>
  </div>
</div>

<style lang="scss">
  .form-row {
    display: flex;
    gap: 12px;
  }

  .form-grow {
    flex: 1;
  }

  .form-fixed-128 {
    width: 128px;
  }

  .input-row {
    display: flex;
    gap: 8px;
  }

  .btn-text {
    padding: 8px 16px;
    border: none;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    background: transparent;
    cursor: pointer;
  }

  .mode-toggle {
    display: flex;
    gap: 0;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--border);
  }

  .mode-btn {
    flex: 1;
    padding: 8px 16px;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition:
      background-color 0.15s,
      color 0.15s;

    &.active {
      background-color: var(--accent);
      color: white;
    }
  }

  .checkbox-row {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 14px;
    color: var(--text-primary);
  }

  .checkbox {
    width: 16px;
    height: 16px;
    cursor: pointer;
    accent-color: var(--accent);
  }

  .checkbox-hint {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
    margin-bottom: 0;

    code {
      font-family: "Courier New", monospace;
      font-size: 11px;
    }
  }
</style>
