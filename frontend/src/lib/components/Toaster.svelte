<script lang="ts">
  import { toasts, dismissToast } from "../stores/toast";
  import { X } from "@lucide/svelte";

  function iconColor(type: string): string {
    switch (type) {
      case "success":
        return "var(--success)";
      case "info":
        return "var(--accent)";
      default:
        return "var(--danger)";
    }
  }
</script>

<div class="toast-container">
  {#each $toasts as toast (toast.id)}
    <div class="toast toast-{toast.type}">
      <span class="toast-message" style="color: var(--text-primary);">
        {toast.message}
      </span>
      <button
        class="toast-close"
        onclick={() => dismissToast(toast.id)}
        title="Dismiss"
      >
        <X class="icon-sm" style="color: var(--text-muted);" />
      </button>
    </div>
  {/each}
</div>

<style lang="scss">
  .toast-container {
    position: fixed;
    bottom: 16px;
    right: 16px;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    border-radius: 8px;
    background-color: var(--bg-secondary);
    border: 1px solid var(--border);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    min-width: 280px;
    max-width: 420px;
    pointer-events: auto;
    animation: toast-slide-in 0.2s ease-out;
  }

  .toast-error {
    border-left: 3px solid var(--danger);
  }

  .toast-success {
    border-left: 3px solid var(--success);
  }

  .toast-info {
    border-left: 3px solid var(--accent);
  }

  .toast-message {
    font-size: 13px;
    flex: 1;
    line-height: 1.4;
  }

  .toast-close {
    padding: 2px;
    border: none;
    border-radius: 4px;
    background: transparent;
    cursor: pointer;
    flex-shrink: 0;

    &:hover {
      background-color: rgba(0, 0, 0, 0.1);
    }
  }

  @keyframes toast-slide-in {
    from {
      opacity: 0;
      transform: translateX(100%);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }
</style>
