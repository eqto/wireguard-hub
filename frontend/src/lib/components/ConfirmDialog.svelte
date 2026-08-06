<script lang="ts">
  import { AlertTriangle } from "@lucide/svelte";

  let {
    title = "Confirm",
    message,
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    danger = true,
    onConfirm,
    onCancel,
  }: {
    title?: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    danger?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
  } = $props();
</script>

<div
  class="modal-overlay"
  onclick={onCancel}
  onkeydown={(e) => e.key === "Escape" && onCancel()}
  role="button"
  tabindex="0"
>
  <div
    class="modal confirm-modal"
    onclick={(e) => e.stopPropagation()}
    role="dialog"
    tabindex="0"
  >
    <div class="confirm-body">
      <AlertTriangle
        class="icon-lg"
        style="color: {danger ? 'var(--danger)' : 'var(--accent)'};"
      />
      <div>
        <h2 class="confirm-title">{title}</h2>
        <p class="confirm-message" style="color: var(--text-secondary);">
          {message}
        </p>
      </div>
    </div>

    <div class="modal-footer">
      <button
        onclick={onCancel}
        class="btn-text"
        style="color: var(--text-secondary);"
      >
        {cancelLabel}
      </button>
      <button
        onclick={onConfirm}
        class="btn {danger ? 'btn-danger' : 'btn-primary'}"
      >
        {confirmLabel}
      </button>
    </div>
  </div>
</div>

<style lang="scss">
  .confirm-modal {
    max-width: 420px;
  }

  .confirm-body {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 20px;
  }

  .confirm-title {
    font-size: 16px;
    font-weight: 600;
    margin: 0 0 6px 0;
  }

  .confirm-message {
    font-size: 14px;
    margin: 0;
    line-height: 1.4;
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

  .btn-danger {
    background-color: var(--danger);
    color: white;

    &:hover {
      opacity: 0.9;
    }
  }
</style>
