<script lang="ts">
  import { Sun, Moon } from "@lucide/svelte";
  import { theme, toggleTheme } from "../stores/servers";

  let currentTheme = $derived($theme);
</script>

<button
  class="theme-toggle"
  style="background-color: var(--bg-tertiary); border: 1px solid var(--border);"
  onclick={toggleTheme}
  title="Toggle theme"
>
  {#if currentTheme === "dark"}
    <Sun class="icon" style="color: var(--text-secondary);" />
  {:else}
    <Moon class="icon" style="color: var(--text-secondary);" />
  {/if}
</button>

<style lang="scss">
  .theme-toggle {
    position: fixed;
    bottom: 16px;
    right: 16px;
    padding: 10px;
    border-radius: 8px;
    z-index: 50;
    transition: opacity 0.15s;

    &:hover {
      opacity: 0.85;
    }
  }
</style>
