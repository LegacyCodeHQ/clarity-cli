<script lang="ts">
  import { cn } from "$lib/utils";
  import Tooltip from "$lib/components/ui/tooltip.svelte";

  interface Props {
    connected: boolean;
  }

  let { connected }: Props = $props();

  const statusText = $derived(connected ? "Connected" : "Reconnecting…");
</script>

<Tooltip text={statusText}>
  <span role="status" aria-label={statusText} class="relative flex size-4 items-center justify-center">
    <span class={cn("absolute inset-0 rounded-full opacity-20", connected ? "bg-connected" : "bg-reconnecting")}></span>
    <span class={cn("size-2 rounded-full", connected ? "bg-connected" : "bg-reconnecting")}></span>
  </span>
</Tooltip>
