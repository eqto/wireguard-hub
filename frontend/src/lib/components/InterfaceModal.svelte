<script lang="ts">
  import { X, Loader2, Sparkles } from "@lucide/svelte";
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
  let listenPort = $state(51820);
  let privateKey = $state("");
  let address = $state("10.0.0.1/24");
  let endpoint = $state("");
  let generating = $state(false);
  let creating = $state(false);

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
        listenPort: Number(listenPort),
        privateKey: privateKey,
        address: address,
        endpoint: endpoint,
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
        <div class="form-group form-fixed-128">
          <label class="label">Listen Port</label>
          <input
            bind:value={listenPort}
            type="number"
            placeholder="51820"
            class="input"
          />
        </div>
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
              <Loader2 class="icon spin" />
            {:else}
              <Sparkles class="icon" />
            {/if}
            Generate
          </button>
        </div>
      </div>

      <div class="form-group">
        <label class="label">Endpoint (optional)</label>
        <input
          bind:value={endpoint}
          type="text"
          placeholder="vpn.example.com:51820"
          class="input"
        />
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
        disabled={creating || !name || !listenPort}
        class="btn btn-primary"
      >
        {#if creating}
          <Loader2 class="icon spin" />
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
</style>
