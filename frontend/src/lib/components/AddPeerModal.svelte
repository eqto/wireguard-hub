<script lang="ts">
  import { X, Loader2, Sparkles, Copy, Check } from "@lucide/svelte";
  import * as WireguardService from "../../../bindings/wireguardadmin/internal/wireguard/service.js";
  import { error } from "../stores/servers";
  import { unwrapResponse } from "../utils";

  let {
    serverId,
    interfaceName,
    onClose,
  }: {
    serverId: string;
    interfaceName: string;
    onClose: () => void;
  } = $props();

  let publicKey = $state("");
  let allowedIPs = $state("10.0.0.2/32");
  let presharedKey = $state("");
  let endpoint = $state("");
  let persistentKeepalive = $state(0);
  let peerName = $state("");
  let peerDescription = $state("");
  let generating = $state(false);
  let adding = $state(false);
  let result = $state<{ publicKey: string; config: string } | null>(null);
  let copied = $state(false);

  async function handleGenerate() {
    generating = true;
    try {
      const res = await WireguardService.GenerateKeyPair(serverId);
      const kp = unwrapResponse(res);
      publicKey = kp.publicKey;
    } catch (e: any) {
      error.set(e?.message || String(e));
    } finally {
      generating = false;
    }
  }

  async function handleAdd() {
    adding = true;
    try {
      const req = {
        serverId: serverId,
        interface: interfaceName,
        publicKey: publicKey,
        allowedIPs: allowedIPs
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
        presharedKey: presharedKey,
        endpoint: endpoint,
        persistentKeepalive: Number(persistentKeepalive),
        name: peerName,
        description: peerDescription,
      };
      const res = await WireguardService.AddPeer(req);
      result = unwrapResponse(res);
    } catch (e: any) {
      error.set(e?.message || String(e));
    } finally {
      adding = false;
    }
  }

  function copyConfig() {
    if (result?.config) {
      navigator.clipboard.writeText(result.config);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    }
  }
</script>

<div
  class="modal-overlay"
  on:click={onClose}
  on:keydown={(e) => e.key === "Escape" && onClose()}
  role="button"
  tabindex="0"
>
  <div
    class="modal"
    on:click={(e) => e.stopPropagation()}
    role="dialog"
    tabindex="0"
  >
    <div class="modal-header">
      <h2 class="modal-title">Add Peer to {interfaceName}</h2>
      <button on:click={onClose} class="close-btn">
        <X class="icon-lg" style="color: var(--text-secondary);" />
      </button>
    </div>

    {#if result}
      <div class="modal-body">
        <p class="peer-result-text" style="color: var(--text-secondary);">
          Peer added successfully! Client config:
        </p>
        <div class="peer-config-wrap">
          <pre
            class="peer-config"
            style="background-color: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border);">{result.config}</pre>
          <button
            on:click={copyConfig}
            class="peer-copy-btn"
            style="background-color: var(--bg-secondary); border: 1px solid var(--border);"
            title="Copy"
          >
            {#if copied}
              <Check class="icon-sm" style="color: var(--success);" />
            {:else}
              <Copy class="icon-sm" style="color: var(--text-secondary);" />
            {/if}
          </button>
        </div>
        <div class="peer-config-footer">
          <button on:click={onClose} class="btn btn-primary"> Done </button>
        </div>
      </div>
    {:else}
      <div class="modal-body">
        <div class="form-group">
          <label class="label">Name (optional)</label>
          <input
            bind:value={peerName}
            type="text"
            placeholder="e.g. Alice's laptop"
            class="input"
          />
        </div>

        <div class="form-group">
          <label class="label">Description (optional)</label>
          <input
            bind:value={peerDescription}
            type="text"
            placeholder="e.g. Work laptop for remote access"
            class="input"
          />
        </div>

        <div class="form-group">
          <label class="label">Public Key</label>
          <div class="input-row">
            <input
              bind:value={publicKey}
              type="text"
              placeholder="Auto-generated or paste existing"
              class="input input-mono"
              style="flex: 1;"
            />
            <button
              on:click={handleGenerate}
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
          <label class="label">Allowed IPs (comma-separated)</label>
          <input
            bind:value={allowedIPs}
            type="text"
            placeholder="10.0.0.2/32"
            class="input"
          />
        </div>

        <div class="form-group">
          <label class="label">Preshared Key (optional)</label>
          <input
            bind:value={presharedKey}
            type="text"
            placeholder="Leave empty for none"
            class="input input-mono"
          />
        </div>

        <div class="form-row">
          <div class="form-group form-grow">
            <label class="label">Endpoint (optional)</label>
            <input
              bind:value={endpoint}
              type="text"
              placeholder="peer.example.com:51820"
              class="input"
            />
          </div>
          <div class="form-group form-fixed-128">
            <label class="label">Keepalive</label>
            <input
              bind:value={persistentKeepalive}
              type="number"
              placeholder="0"
              class="input"
            />
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button
          on:click={onClose}
          class="btn-text"
          style="color: var(--text-secondary);"
        >
          Cancel
        </button>
        <button
          on:click={handleAdd}
          disabled={adding || !publicKey || !allowedIPs}
          class="btn btn-primary"
        >
          {#if adding}
            <Loader2 class="icon spin" />
          {/if}
          Add Peer
        </button>
      </div>
    {/if}
  </div>
</div>

<style lang="scss">
  .peer-result-text {
    font-size: 14px;
    margin-bottom: 12px;
  }

  .peer-config-wrap {
    position: relative;

    .peer-config {
      padding: 12px;
      border-radius: 8px;
      font-size: 12px;
      font-family: "Courier New", monospace;
      overflow-x: auto;
      max-height: 240px;
      white-space: pre-wrap;
    }

    .peer-copy-btn {
      position: absolute;
      top: 8px;
      right: 8px;
      padding: 6px;
      border: none;
      border-radius: 8px;
      cursor: pointer;
    }
  }

  .peer-config-footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
  }

  .input-row {
    display: flex;
    gap: 8px;
  }

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
