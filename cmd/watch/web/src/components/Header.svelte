<script lang="ts">
  import SourceSelector from './SourceSelector.svelte';
  import Badge from '../lib/components/ui/badge.svelte';

  interface Props {
    pageTitle: string;
    connected: boolean;
  }

  let { pageTitle, connected }: Props = $props();

  const statusText = $derived(
    connected ? 'Connected' : 'Reconnecting'
  );

  const statusVariant = $derived(
    connected ? 'success' : 'destructive'
  );
</script>

<div class="px-4 py-2.5 bg-card border-b border-border flex items-center gap-4">
  <h1 class="text-sm font-semibold text-foreground">{pageTitle}</h1>
  <span class="flex-1"></span>
  <SourceSelector />
  <Badge variant={statusVariant} class="gap-1.5 transition-all">
    <span class="w-2 h-2 rounded-full {connected ? 'bg-white shadow-[0_0_4px_rgba(255,255,255,0.5)]' : 'bg-white/80'} transition-all"></span>
    <span>{statusText}</span>
  </Badge>
</div>
