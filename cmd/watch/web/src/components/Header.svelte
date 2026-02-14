<script lang="ts">
  import SourceSelector from './SourceSelector.svelte';
  import Badge from '../lib/components/ui/badge.svelte';

  interface Props {
    pageTitle: string;
    connected: boolean;
  }

  let { pageTitle, connected }: Props = $props();

  const statusText = $derived(
    connected ? 'Connected' : 'Reconnecting...'
  );
</script>

<div class="px-4 py-1.5 bg-card border-b border-border flex items-center gap-4 text-xs">
  <h1 class="font-medium text-foreground">{pageTitle}</h1>
  <div class="flex items-center gap-1.5 text-muted-foreground">
    <span class="w-1.5 h-1.5 rounded-full {connected ? 'bg-green-500' : 'bg-red-400'}"></span>
    <span>{statusText}</span>
  </div>
  <span class="flex-1"></span>
  <div class="flex items-center gap-2">
    <span class="text-muted-foreground">View</span>
    <SourceSelector />
  </div>
</div>
