<script lang="ts">
  import { X, LoaderCircle } from "@lucide/svelte";
  import { untrack } from "svelte";
  import * as WireguardService from "../../../bindings/wireguardhub/internal/wireguard/service.js";
  import { error } from "../stores/servers";
  import { unwrapResponse } from "../utils";

  let {
    serverId,
    interfaceName,
    peer,
    isClientInterface = false,
    onClose,
    onSaved,
  }: {
    serverId: string;
    interfaceName: string;
    peer: any;
    isClientInterface?: boolean;
    onClose: () => void;
    onSaved: () => void;
  } = $props();

  let name = $state(untrack(() => peer?.name || ""));
  let description = $state(untrack(() => peer?.description || ""));
  let endpoint = $state(untrack(() => peer?.endpoint || ""));
  let allowedIPs = $state(untrack(() => (peer?.allowedIPs || []).join(", ")));
  let publicKey = $state(untrack(() => peer?.publicKey || ""));
  let saving = $state(false);

  async function handleSave() {
    saving = true;
    try {
      const req: any = {
        serverId: serverId,
        interface: interfaceName,
        publicKey: peer.publicKey,
        name: name,
        description: description,
      };
      if (isClientInterface) {
        if (publicKey !== peer.publicKey) {
          req.newPublicKey = publicKey;
        }
        if (endpoint) req.endpoint = endpoint;
        if (allowedIPs.trim()) {
          req.allowedIPs = allowedIPs
            .split(",")
            .map((s: string) => s.trim())
            .filter((s: string) => s);
        }
        req.restart = true;
      }
      const res = await WireguardService.UpdatePeerMeta(req);
      unwrapResponse(res);
      onSaved();
      onClose();
    } catch (e: any) {
      error.set(e?.message || String(e));
    } finally {
      saving = false;
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
    onkeydown={(e) => e.stopPropagation()}
    role="dialog"
    tabindex="0"
  >
    <div class="modal-header">
      <h2 class="modal-title">
        {isClientInterface ? "Server Configuration" : "Edit Peer"}
      </h2>
      <button onclick={onClose} class="close-btn">
        <X class="icon-lg" style="color: var(--text-secondary);" />
      </button>
    </div>

    <div class="modal-body">
      {#if isClientInterface}
        <div class="form-group">
          <label class="label" for="edit-peer-public-key">Public Key</label>
          <input
            id="edit-peer-public-key"
            bind:value={publicKey}
            type="text"
            placeholder="(none)"
            class="input input-mono"
          />
        </div>
      {:else}
        <div class="peer-key-info" style="color: var(--text-muted);">
          <span class="peer-key-label">Public Key:</span>
          <span class="peer-key-value">{publicKey || "(none)"}</span>
        </div>
      {/if}

      {#if isClientInterface}
        <div class="form-group">
          <label class="label" for="edit-peer-endpoint">Server Address</label>
          <input
            id="edit-peer-endpoint"
            bind:value={endpoint}
            type="text"
            placeholder="vpn.example.com:51820"
            class="input"
          />
        </div>

        <div class="form-group">
          <label class="label" for="edit-peer-allowed-ips">Allowed IPs</label>
          <input
            id="edit-peer-allowed-ips"
            bind:value={allowedIPs}
            type="text"
            placeholder="0.0.0.0/0, ::/0"
            class="input"
          />
        </div>
      {/if}

      {#if !isClientInterface}
        <div class="form-group">
          <label class="label" for="edit-peer-name">Name</label>
          <input
            id="edit-peer-name"
            bind:value={name}
            type="text"
            placeholder="e.g. Alice's laptop"
            class="input"
          />
        </div>

        <div class="form-group">
          <label class="label" for="edit-peer-description">Description</label>
          <input
            id="edit-peer-description"
            bind:value={description}
            type="text"
            placeholder="e.g. Work laptop for remote access"
            class="input"
          />
        </div>
      {/if}
    </div>

    <div class="modal-footer">
      <button
        onclick={onClose}
        class="btn-text"
        style="color: var(--text-secondary);"
      >
        Cancel
      </button>
      <button onclick={handleSave} disabled={saving} class="btn btn-primary">
        {#if saving}
          <LoaderCircle class="icon spin" />
        {/if}
        Save
      </button>
    </div>
  </div>
</div>

<style lang="scss">
  .peer-key-info {
    font-size: 12px;
    margin-bottom: 16px;

    .peer-key-label {
      margin-right: 4px;
    }

    .peer-key-value {
      font-family: "Courier New", monospace;
    }
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
