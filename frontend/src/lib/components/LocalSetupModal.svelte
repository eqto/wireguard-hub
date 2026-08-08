<script lang="ts">
  import { onMount } from "svelte";
  import { X, Plug, Loader2 } from "@lucide/svelte";
  import * as ServerService from "../../../bindings/wireguardhub/internal/server/service.js";
  import * as WireguardService from "../../../bindings/wireguardhub/internal/wireguard/service.js";
  import { unwrapResponse } from "../utils";

  let {
    onSave,
    onConfigured,
    onClose,
  }: {
    onSave: () => void;
    onConfigured: () => void;
    onClose: () => void;
  } = $props();

  let username = $state("");
  let password = $state("");
  let savePassword = $state(false);
  let testing = $state(false);
  let saving = $state(false);
  let testResult = $state<{ success: boolean; message: string } | null>(null);
  let configured = $state(false);

  onMount(async () => {
    try {
      const result = await ServerService.GetLocalConfig();
      const cfg = unwrapResponse(result);
      if (cfg) {
        username = cfg.username || "";
        configured = cfg.configured ?? false;
      }
    } catch {
      // ignore
    }
  });

  async function handleTest() {
    testing = true;
    testResult = null;
    try {
      if (username || password) {
        await ServerService.SetLocalSessionCredentials(username, password);
      }
      const data = {
        id: "local",
        name: "Local",
        host: "localhost",
        isLocal: true,
      };
      const result = await ServerService.TestConnection(data);
      const r = unwrapResponse(result);
      testResult = { success: r.success, message: r.message };
    } catch (e: any) {
      testResult = { success: false, message: e?.message || String(e) };
    } finally {
      testing = false;
    }
  }

  async function handleSave() {
    saving = true;
    testResult = null;
    try {
      await ServerService.SetLocalSessionCredentials(username, password);
      const result = await WireguardService.GetStatus("local");
      unwrapResponse(result);
      if (savePassword) {
        await ServerService.SaveLocalConfig({ username, password });
      }
      onSave();
      onConfigured();
    } catch (e: any) {
      const msg = e?.message || String(e);
      testResult = { success: false, message: msg };
      try {
        await ServerService.ClearLocalSessionCredentials();
      } catch {
        // ignore
      }
    } finally {
      saving = false;
    }
  }
</script>

<div
  class="modal-overlay"
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
      <h2 class="modal-title">Local Management Setup</h2>
      <button onclick={onClose} class="close-btn">
        <X class="icon-lg" style="color: var(--text-secondary);" />
      </button>
    </div>

    <div class="modal-body">
      <p class="modal-description" style="color: var(--text-muted);">
        Configure sudo credentials for managing WireGuard on this machine. This
        is optional — leave empty if you have passwordless sudo.
      </p>

      <div class="form-group">
        <label class="label">Sudo Username</label>
        <input
          bind:value={username}
          type="text"
          placeholder="your-username"
          class="input"
        />
      </div>

      <div class="form-group">
        <label class="label">Sudo Password</label>
        <input
          bind:value={password}
          type="password"
          placeholder="••••••••"
          class="input"
          onkeydown={(e) => e.key === "Enter" && handleSave()}
        />
      </div>

      <div class="form-group">
        <label class="checkbox-label">
          <input type="checkbox" bind:checked={savePassword} />
          <span style="color: var(--text-primary);">Save password</span>
        </label>
      </div>

      {#if testResult}
        <div
          class="test-result"
          style={testResult.success
            ? "background-color: rgba(22,163,74,0.1); color: var(--success);"
            : "background-color: rgba(220,38,38,0.1); color: var(--danger);"}
        >
          {testResult.message}
        </div>
      {/if}
    </div>

    <div class="modal-footer">
      <button onclick={handleTest} disabled={testing} class="btn btn-secondary">
        {#if testing}
          <Loader2 class="icon spin" />
        {:else}
          <Plug class="icon" />
        {/if}
        Test Connection
      </button>
      <div class="footer-actions">
        <button
          onclick={onClose}
          class="btn-text"
          style="color: var(--text-secondary);"
        >
          Cancel
        </button>
        <button onclick={handleSave} disabled={saving} class="btn btn-primary">
          {#if saving}
            <Loader2 class="icon spin" />
          {/if}
          Save
        </button>
      </div>
    </div>
  </div>
</div>

<style lang="scss">
  .modal-description {
    font-size: 13px;
    margin-bottom: 16px;
    line-height: 1.5;
  }

  .footer-actions {
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

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 14px;
  }

  .spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
</style>
