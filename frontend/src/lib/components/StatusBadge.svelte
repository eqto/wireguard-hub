<script lang="ts">
  import { statusColor } from "../utils";

  let { status }: { status: string } = $props();

  let color = $derived(statusColor(status));
  let label = $derived(
    status === "connected"
      ? "Connected"
      : status === "offline"
        ? "Offline"
        : "Untested",
  );
</script>

<div class="status-badge">
  <div class="status-dot" style="background-color: {color};"></div>
  <span class="status-label" style="color: var(--text-secondary);">{label}</span
  >
</div>

<style lang="scss">
  .status-badge {
    display: flex;
    align-items: center;
    gap: 8px;

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
    }

    .status-label {
      font-size: 12px;
      font-weight: 500;
    }
  }
</style>
