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

<div class="px-4 py-2.5 bg-card border-t border-border flex items-center gap-4">
  <span class="text-xs font-medium text-muted-foreground w-16">{$viewModel.timeline.modeText}</span>
  <input
    type="range"
    class="timeline-slider flex-1 min-w-[120px] cursor-pointer"
    min="0"
    max={$viewModel.timeline.sliderMax}
    value={$viewModel.timeline.sliderValue}
    disabled={$viewModel.timeline.sliderDisabled}
    oninput={handleSliderInput}
  />
  <Button
    variant="ghost"
    size="sm"
    disabled={$viewModel.timeline.liveButtonDisabled}
    onclick={handleJumpToLatest}
    class="text-xs"
  >
    Live
  </Button>
  <span class="min-w-[100px] text-right text-xs text-muted-foreground">{$viewModel.timeline.metaText}</span>
</div>
