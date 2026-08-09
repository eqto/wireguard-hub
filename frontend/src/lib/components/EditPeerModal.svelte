<script lang="ts">
  import { X, LoaderCircle, Plus } from "@lucide/svelte";
  import { untrack } from "svelte";
  import * as WireguardService from "../../../bindings/wireguardhub/internal/wireguard/service.js";
  import { error } from "../stores/servers";
  import { unwrapResponse, sortIPs } from "../utils";

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
  let allowedIPs = $state<string[]>(untrack(() => peer?.allowedIPs || []));
  let publicKey = $state(untrack(() => peer?.publicKey || ""));
  let saving = $state(false);
  let newIP = $state("");

  function addIP() {
    const ip = newIP.trim();
    if (ip && !allowedIPs.includes(ip)) {
      allowedIPs = [...allowedIPs, ip];
    }
    newIP = "";
  }

  function removeIP(ip: string) {
    allowedIPs = allowedIPs.filter((x) => x !== ip);
  }

  function handleIPKeydown(e: KeyboardEvent) {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      addIP();
    }
  }

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
        if (allowedIPs.length > 0) {
          req.allowedIPs = allowedIPs;
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
          <div class="chips-container">
            {#each sortIPs(allowedIPs) as ip (ip)}
              <span class="ip-chip">
                <button
                  type="button"
                  class="chip-remove"
                  onclick={() => removeIP(ip)}
                  title="Remove"
                >
                  <X class="chip-x-icon" />
                </button>
                {ip}
              </span>
            {/each}
            <div class="chip-input-wrap">
              <input
                id="edit-peer-allowed-ips"
                bind:value={newIP}
                onkeydown={handleIPKeydown}
                type="text"
                placeholder={allowedIPs.length === 0
                  ? "0.0.0.0/0, ::/0"
                  : "Add IP…"}
                class="input chip-input"
              />
              <button
                type="button"
                onclick={addIP}
                disabled={!newIP.trim()}
                class="chip-add-btn"
                title="Add"
              >
                <Plus class="chip-add-icon" />
              </button>
            </div>
          </div>
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

  .chips-container {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding: 8px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background-color: var(--bg-secondary);
    align-items: center;
  }

  .ip-chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 4px 2px 2px;
    border-radius: 4px;
    font-size: 12px;
    font-family: "Courier New", monospace;
    background-color: color-mix(in srgb, var(--accent) 12%, transparent);
    border: 1px solid color-mix(in srgb, var(--accent) 30%, transparent);
    color: var(--text-primary);
    white-space: nowrap;
  }

  .chip-remove {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 2px;
    border-radius: 3px;
    color: var(--text-secondary);
    transition:
      color 0.15s,
      background-color 0.15s;

    &:hover {
      color: var(--danger, #e5484d);
      background-color: color-mix(
        in srgb,
        var(--danger, #e5484d) 15%,
        transparent
      );
    }
  }

  .chip-x-icon {
    width: 12px;
    height: 12px;
  }

  .chip-input-wrap {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
    min-width: 120px;
  }

  .chip-input {
    flex: 1;
    min-width: 80px;
    padding: 4px 8px;
    font-size: 12px;
    border: none;
    background: transparent;
    color: var(--text-primary);
    outline: none;

    &::placeholder {
      color: var(--text-muted);
    }
  }

  .chip-add-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    color: var(--text-secondary);
    transition:
      color 0.15s,
      background-color 0.15s;

    &:hover:not(:disabled) {
      color: var(--accent);
      background-color: color-mix(in srgb, var(--accent) 15%, transparent);
    }

    &:disabled {
      opacity: 0.4;
      cursor: default;
    }
  }

  .chip-add-icon {
    width: 16px;
    height: 16px;
  }
</style>
