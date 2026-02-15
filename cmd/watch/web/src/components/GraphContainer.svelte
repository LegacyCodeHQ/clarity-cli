<script lang="ts">
  import { onMount } from 'svelte';
  import { viewModel } from '../lib/stores/graphStore';
  import { initGraphviz, renderDot } from '../lib/graphviz';
  import Skeleton from '../lib/components/ui/skeleton.svelte';

  let graphContainer: HTMLDivElement;
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
    if (!graphvizReady || !graphContainer) return;

    try {
      const svg = await renderDot(dot);
      graphContainer.innerHTML = svg;
      renderError = null;
    } catch (err) {
      console.error('Graphviz render error:', err);
      renderError = 'Render error';
    }
  }

  $effect(() => {
    if ($viewModel.renderDot && graphvizReady) {
      renderGraph($viewModel.renderDot);
    } else if (!$viewModel.renderDot && graphContainer) {
      graphContainer.innerHTML = '';
    }
  });
</script>

<div class="flex-1 overflow-auto bg-background">
  <div class="h-full flex items-center justify-center bg-[#2a2a2a] shadow-[inset_0_2px_8px_rgba(0,0,0,0.3)] [&_svg]:max-w-full [&_svg]:max-h-full relative">
    <!-- Graph rendering container (DOM manipulated) -->
    <div bind:this={graphContainer} class="w-full h-full flex items-center justify-center p-12 transition-opacity duration-300 [&_svg]:transition-all [&_svg]:duration-300"></div>

    <!-- Message container (Svelte managed) -->
    {#if !graphvizReady}
      <div class="absolute inset-0 flex items-center justify-center">
        <div class="flex flex-col items-center gap-4 animate-fade-in">
          <Skeleton class="h-24 w-48" />
          <p class="text-muted-foreground text-sm">Loading Graphviz...</p>
        </div>
      </div>
    {:else if renderError}
      <div class="absolute inset-0 flex items-center justify-center">
        <p class="text-destructive text-sm font-medium">{renderError}</p>
      </div>
    {:else if !$viewModel.renderDot}
      <div class="absolute inset-0 flex items-center justify-center">
        <div class="flex flex-col items-center gap-3 text-center animate-fade-in">
          <svg class="w-16 h-16 text-muted-foreground/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <div>
            <p class="text-muted-foreground font-medium mb-1">Waiting for changes</p>
            <p class="text-muted-foreground/60 text-xs max-w-xs">Make changes to your files to see the dependency graph appear here</p>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
