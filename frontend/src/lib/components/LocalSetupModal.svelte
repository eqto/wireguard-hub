<script lang="ts">
  import { onMount } from "svelte";
  import { X, Loader2 } from "@lucide/svelte";
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
  let saving = $state(false);
  let testResult = $state<{ success: boolean; message: string } | null>(null);
  let configured = $state(false);
  let usernameInput = $state<HTMLInputElement | null>(null);

  onMount(async () => {
    usernameInput?.focus();
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

  async function handleSave() {
    if (!password) return;
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
        Configure sudo credentials for managing WireGuard on this machine.
      </p>

      <div class="form-group">
        <label class="label">Sudo Username</label>
        <input
          bind:this={usernameInput}
          bind:value={username}
          type="text"
          class="input"
        />
      </div>

      <div class="form-group">
        <label class="label">Sudo Password</label>
        <input
          bind:value={password}
          type="password"
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
      <div class="footer-actions">
        <button
          onclick={onClose}
          class="btn-text"
          style="color: var(--text-secondary);"
        >
          Cancel
        </button>
        <button
          onclick={handleSave}
          disabled={saving || !password}
          class="btn btn-primary"
        >
          {#if saving}
            <Loader2 class="icon spin" />
          {/if}
          Open
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

  .test-result {
    padding: 12px;
    border-radius: 8px;
    font-size: 13px;
    margin-top: 8px;
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
