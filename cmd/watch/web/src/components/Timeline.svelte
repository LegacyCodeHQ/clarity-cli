<script lang="ts">
  import { viewModel, graphStore } from '../lib/stores/graphStore';
  import Button from '../lib/components/ui/button.svelte';

  function handleSliderInput(event: Event) {
    const target = event.target as HTMLInputElement;
    graphStore.onSliderInput(target.value);
  }

  function handleJumpToLatest() {
    graphStore.onJumpToLatest();
  }
</script>

<div class="px-4 py-1.5 bg-card border-t border-border flex items-center gap-3 text-xs">
  <span class="text-muted-foreground w-16">{$viewModel.timeline.modeText}</span>
  <input
    type="range"
    class="flex-1 min-w-[120px] accent-primary h-1 cursor-pointer"
    min="0"
    max={$viewModel.timeline.sliderMax}
    value={$viewModel.timeline.sliderValue}
    disabled={$viewModel.timeline.sliderDisabled}
    oninput={handleSliderInput}
  />
  <button
    class="px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
    disabled={$viewModel.timeline.liveButtonDisabled}
    onclick={handleJumpToLatest}
  >
    Jump to latest
  </button>
  <span class="min-w-[100px] text-right text-muted-foreground">{$viewModel.timeline.metaText}</span>
</div>
