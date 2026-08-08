<script lang="ts">
  import { X, Plug, Loader2 } from "@lucide/svelte";
  import * as ServerService from "../../../bindings/wireguardhub/internal/server/service.js";
  import { error, servers } from "../stores/servers";
  import { unwrapResponse } from "../utils";

  let {
    server = null,
    onSave,
    onClose,
  }: {
    server?: any;
    onSave: (data: any) => void;
    onClose: () => void;
  } = $props();

  let name = $state(server?.name || "");
  let host = $state(server?.host || "");
  let port = $state(server?.port || 22);
  let username = $state(server?.username || "root");
  let authMethod = $state(server?.authMethod || "password");
  let password = $state(server?.password || "");
  let privateKey = $state(server?.privateKey || "");
  let passphrase = $state(server?.passphrase || "");
  let viaServerId = $state(server?.viaServerId || "");
  let testing = $state(false);
  let testResult = $state<{ success: boolean; message: string } | null>(null);

  // Candidates for jump host: exclude the server being edited and servers that
  // themselves use a jump host (single-hop constraint). Backend still validates.
  let jumpCandidates = $derived(
    ($servers || []).filter((s) => s.id !== server?.id && !s.viaServerId),
  );

  async function handleTest() {
    testing = true;
    testResult = null;
    try {
      const data = {
        id: server?.id || "",
        name: name,
        host: host,
        port: Number(port),
        username: username,
        authMethod: authMethod,
        password: password,
        privateKey: privateKey,
        passphrase: passphrase,
        viaServerId: viaServerId,
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

  function handleSave() {
    if (!name || !host || !username) return;
    onSave({
      id: server?.id || "",
      name: name,
      host: host,
      port: Number(port),
      username: username,
      authMethod: authMethod,
      password: password,
      privateKey: privateKey,
      passphrase: passphrase,
      viaServerId: viaServerId,
    });
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
      <h2 class="modal-title">
        {server ? "Edit Server" : "Add Server"}
      </h2>
      <button onclick={onClose} class="close-btn">
        <X class="icon-lg" style="color: var(--text-secondary);" />
      </button>
    </div>

    <div class="modal-body">
      <div class="form-group">
        <label class="label">Name</label>
        <input
          bind:value={name}
          type="text"
          placeholder="My VPN Server"
          class="input"
        />
      </div>

      <div class="form-row">
        <div class="form-group form-grow">
          <label class="label">Hostname or IP Address</label>
          <input
            bind:value={host}
            type="text"
            placeholder="192.168.1.1"
            class="input"
          />
        </div>
        <div class="form-group form-fixed-96">
          <label class="label">Port</label>
          <input
            bind:value={port}
            type="number"
            placeholder="22"
            class="input"
          />
        </div>
      </div>

      {#if jumpCandidates.length > 0}
        <div class="form-group">
          <label class="label">Connect via (optional)</label>
          <select bind:value={viaServerId} class="input">
            <option value="">Direct connection</option>
            {#each jumpCandidates as candidate (candidate.id)}
              <option value={candidate.id}>{candidate.name}</option>
            {/each}
          </select>
        </div>
      {/if}

      <div class="form-group">
        <label class="label">Username</label>
        <input
          bind:value={username}
          type="text"
          placeholder="root"
          class="input"
        />
      </div>

      <div class="form-group">
        <label class="label">Auth Method</label>
        <div class="radio-row">
          <label class="radio-label">
            <input type="radio" bind:group={authMethod} value="password" />
            <span style="color: var(--text-primary);">Password</span>
          </label>
          <label class="radio-label">
            <input type="radio" bind:group={authMethod} value="key" />
            <span style="color: var(--text-primary);">Private Key</span>
          </label>
        </div>
      </div>

      {#if authMethod === "password"}
        <div class="form-group">
          <label class="label">Password</label>
          <input
            bind:value={password}
            type="password"
            placeholder="••••••••"
            class="input"
            onkeydown={(e) => e.key === "Enter" && handleSave()}
          />
        </div>
      {:else}
        <div class="form-group">
          <label class="label">Private Key</label>
          <textarea
            bind:value={privateKey}
            placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
            rows="5"
            class="input input-mono"
          ></textarea>
        </div>
        <div class="form-group">
          <label class="label">Passphrase (optional)</label>
          <input
            bind:value={passphrase}
            type="password"
            placeholder="••••••••"
            class="input"
            onkeydown={(e) => e.key === "Enter" && handleSave()}
          />
        </div>
      {/if}

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
      <button
        onclick={handleTest}
        disabled={testing || !host || !username}
        class="btn btn-secondary"
      >
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
        <button
          onclick={handleSave}
          disabled={!name || !host || !username}
          class="btn btn-primary"
        >
          {server ? "Update" : "Add"}
        </button>
      </div>
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

  .form-fixed-96 {
    width: 96px;

    input[type="number"]::-webkit-inner-spin-button,
    input[type="number"]::-webkit-outer-spin-button {
      -webkit-appearance: none;
      margin: 0;
    }

    input[type="number"] {
      -moz-appearance: textfield;
    }
  }

  .radio-row {
    display: flex;
    gap: 16px;
  }

  .radio-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 14px;
  }

  .test-result {
    padding: 8px 12px;
    border-radius: 8px;
    font-size: 14px;
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
</style>
