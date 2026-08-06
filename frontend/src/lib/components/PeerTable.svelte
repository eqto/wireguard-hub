<script lang="ts">
  import { Trash2, Pencil } from "@lucide/svelte";
  import { formatBytes, formatRelativeTime, truncateKey } from "../utils";

  let {
    peers,
    onRemove,
    onEdit,
  }: { peers: any[]; onRemove: (pubKey: string) => void; onEdit: (peer: any) => void } = $props();
</script>

{#if peers.length === 0}
  <div class="peer-empty">
    <p class="peer-empty-text" style="color: var(--text-muted);">
      No peers connected
    </p>
  </div>
{:else}
  <div class="peer-table-wrap" style="border: 1px solid var(--border);">
    <table class="peer-table">
      <thead>
        <tr
          style="background-color: var(--bg-tertiary); border-bottom: 1px solid var(--border);"
        >
          <th class="peer-th" style="color: var(--text-secondary);">Name</th>
          <th class="peer-th" style="color: var(--text-secondary);"
            >Public Key</th
          >
          <th class="peer-th" style="color: var(--text-secondary);">Endpoint</th
          >
          <th class="peer-th" style="color: var(--text-secondary);"
            >Allowed IPs</th
          >
          <th class="peer-th" style="color: var(--text-secondary);"
            >Handshake</th
          >
          <th
            class="peer-th peer-th-right"
            style="color: var(--text-secondary);">RX</th
          >
          <th
            class="peer-th peer-th-right"
            style="color: var(--text-secondary);">TX</th
          >
          <th class="peer-th"></th>
        </tr>
      </thead>
      <tbody>
        {#each peers as peer (peer.publicKey)}
          <tr class="peer-tr" style="border-bottom: 1px solid var(--border);">
            <td
              class="peer-td"
              style="color: var(--text-primary);"
              title={peer.description || ""}
            >
              {peer.name || truncateKey(peer.publicKey, 12)}
            </td>
            <td
              class="peer-td peer-td-mono"
              style="color: var(--text-secondary);"
              title={peer.publicKey}
            >
              {truncateKey(peer.publicKey, 20)}
            </td>
            <td
              class="peer-td peer-td-xs"
              style="color: var(--text-secondary);"
            >
              {peer.endpoint || "-"}
            </td>
            <td
              class="peer-td peer-td-xs"
              style="color: var(--text-secondary);"
            >
              {(peer.allowedIPs || []).join(", ") || "-"}
            </td>
            <td
              class="peer-td peer-td-xs"
              style="color: var(--text-secondary);"
            >
              {formatRelativeTime(peer.latestHandshake)}
            </td>
            <td
              class="peer-td peer-td-right peer-td-xs"
              style="color: var(--text-secondary);"
            >
              {formatBytes(peer.rxBytes)}
            </td>
            <td
              class="peer-td peer-td-right peer-td-xs"
              style="color: var(--text-secondary);"
            >
              {formatBytes(peer.txBytes)}
            </td>
            <td class="peer-td peer-td-right">
              <div class="peer-actions">
                <button
                  onclick={() => onEdit(peer)}
                  class="peer-remove-btn"
                  title="Edit peer metadata"
                >
                  <Pencil class="icon-sm" style="color: var(--text-secondary);" />
                </button>
                <button
                  onclick={() => onRemove(peer.publicKey)}
                  class="peer-remove-btn"
                  title="Remove peer"
                >
                  <Trash2 class="icon-sm" style="color: var(--danger);" />
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<style lang="scss">
  .peer-empty {
    padding: 24px 0;
    text-align: center;

    &-text {
      font-size: 14px;
    }
  }

  .peer-table-wrap {
    overflow-x: auto;
    border-radius: 8px;
  }

  .peer-table {
    width: 100%;
    font-size: 14px;
    border-collapse: collapse;

    .peer-th {
      text-align: left;
      padding: 10px 16px;
      font-weight: 500;

      &-right {
        text-align: right;
      }
    }

    .peer-tr:hover {
      background-color: rgba(0, 0, 0, 0.05);
    }

    .peer-td {
      padding: 10px 16px;

      &-mono {
        font-family: "Courier New", monospace;
        font-size: 12px;
      }

      &-xs {
        font-size: 12px;
      }

      &-right {
        text-align: right;
      }
    }
  }

  .peer-actions {
    display: flex;
    gap: 4px;
    justify-content: flex-end;
  }

  .peer-remove-btn {
    padding: 4px;
    border: none;
    border-radius: 4px;
    background: transparent;
    cursor: pointer;

    &:hover {
      background-color: rgba(0, 0, 0, 0.1);
    }
  }
</style>
