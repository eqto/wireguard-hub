<script lang="ts">
  import { X, Copy, Check } from "@lucide/svelte";

  let {
    name,
    content,
    onClose,
  }: { name: string; content: string; onClose: () => void } = $props();
  let copied = $state(false);

  function copyContent() {
    navigator.clipboard.writeText(content);
    copied = true;
    setTimeout(() => (copied = false), 2000);
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
    class="modal modal-large"
    on:click={(e) => e.stopPropagation()}
    role="dialog"
    tabindex="0"
  >
    <div class="modal-header">
      <h2 class="modal-title">{name}.conf</h2>
      <div class="config-header-actions">
        <button on:click={copyContent} class="btn btn-secondary btn-small">
          {#if copied}
            <Check class="icon" style="color: var(--success);" />
            Copied
          {:else}
            <Copy class="icon" style="color: var(--text-secondary);" />
            Copy
          {/if}
        </button>
        <button on:click={onClose} class="close-btn">
          <X class="icon-lg" style="color: var(--text-secondary);" />
        </button>
      </div>
    </div>

    <div class="modal-body">
      <pre
        class="config-pre"
        style="background-color: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border);">{content}</pre>
    </div>
  </div>
</div>

<style lang="scss">
  .config-header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .config-pre {
    padding: 16px;
    border-radius: 8px;
    font-size: 12px;
    font-family: "Courier New", monospace;
    overflow-x: auto;
    white-space: pre-wrap;
  }
</style>
