<script lang="ts">
  import { X, Loader2 } from "@lucide/svelte";
  import * as WireguardService from "../../../bindings/wireguardhub/internal/wireguard/service.js";
  import { error } from "../stores/servers";
  import { unwrapResponse } from "../utils";

  let {
    serverId,
    interfaceName,
    peer,
    onClose,
    onSaved,
  }: {
    serverId: string;
    interfaceName: string;
    peer: any;
    onClose: () => void;
    onSaved: () => void;
  } = $props();

  let name = $state(peer?.name || "");
  let description = $state(peer?.description || "");
  let saving = $state(false);

  async function handleSave() {
    saving = true;
    try {
      const req = {
        serverId: serverId,
        interface: interfaceName,
        publicKey: peer.publicKey,
        name: name,
        description: description,
      };
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
    role="dialog"
    tabindex="0"
  >
    <div class="modal-header">
      <h2 class="modal-title">Edit Peer</h2>
      <button onclick={onClose} class="close-btn">
        <X class="icon-lg" style="color: var(--text-secondary);" />
      </button>
    </div>

    <div class="modal-body">
      <div class="peer-key-info" style="color: var(--text-muted);">
        <span class="peer-key-label">Public Key:</span>
        <span class="peer-key-value">{peer.publicKey.slice(0, 20)}...</span>
      </div>

      <div class="form-group">
        <label class="label">Name</label>
        <input
          bind:value={name}
          type="text"
          placeholder="e.g. Alice's laptop"
          class="input"
        />
      </div>

      <div class="form-group">
        <label class="label">Description</label>
        <input
          bind:value={description}
          type="text"
          placeholder="e.g. Work laptop for remote access"
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
        onclick={handleSave}
        disabled={saving}
        class="btn btn-primary"
      >
        {#if saving}
          <Loader2 class="icon spin" />
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
