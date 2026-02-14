<script lang="ts">
  import { viewModel, graphStore } from '../lib/stores/graphStore';

  function handleChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    graphStore.onSourceChange(target.value);
  }
</script>

<select
  id="snapshot-source"
  class="bg-input text-foreground border-0 rounded px-2.5 py-1.5 text-xs font-medium focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all cursor-pointer hover:bg-input/80"
  value={$viewModel.sourceValue}
  onchange={handleChange}
>
  {#each $viewModel.sourceOptions as option}
    <option value={option.value}>{option.text}</option>
  {/each}
</select>
