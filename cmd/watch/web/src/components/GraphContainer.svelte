<script lang="ts">
  import { onMount } from 'svelte';
  import { viewModel } from '../lib/stores/graphStore';
  import { initGraphviz, renderDot } from '../lib/graphviz';
  import Card from '../lib/components/ui/card.svelte';
  import Skeleton from '../lib/components/ui/skeleton.svelte';

  let container: HTMLDivElement;
  let graphvizReady = $state(false);
  let renderError = $state<string | null>(null);

  onMount(async () => {
    try {
      await initGraphviz();
      graphvizReady = true;
    } catch (err) {
      console.error('Failed to initialize Graphviz:', err);
      renderError = 'Failed to load Graphviz';
    }
  });

  async function renderGraph(dot: string) {
    if (!graphvizReady || !container) return;

    try {
      const svg = await renderDot(dot);
      container.innerHTML = svg;
      renderError = null;
    } catch (err) {
      console.error('Graphviz render error:', err);
      renderError = 'Render error';
    }
  }

  $effect(() => {
    if ($viewModel.renderDot && graphvizReady) {
      renderGraph($viewModel.renderDot);
    } else if (!$viewModel.renderDot && container) {
      container.innerHTML = '';
    }
  });
</script>

<div class="flex-1 overflow-auto bg-background">
  <div class="h-full flex items-center justify-center bg-[#2a2a2a] shadow-[inset_0_2px_8px_rgba(0,0,0,0.3)] [&_svg]:max-w-full [&_svg]:max-h-full">
    <div bind:this={container} class="w-full h-full flex items-center justify-center p-12 transition-opacity duration-300 [&_svg]:transition-all [&_svg]:duration-300">
      {#if !graphvizReady}
        <div class="flex flex-col items-center gap-4 animate-fade-in">
          <Skeleton class="h-24 w-48" />
          <p class="text-muted-foreground text-sm">Loading Graphviz...</p>
        </div>
      {:else if renderError}
        <p class="text-destructive text-sm font-medium">{renderError}</p>
      {:else if !$viewModel.renderDot}
        <p class="text-muted-foreground text-sm backdrop-blur-sm bg-card/30 px-4 py-2 rounded-md">No uncommitted changes. Waiting for file changes...</p>
      {/if}
    </div>
  </div>
</div>
