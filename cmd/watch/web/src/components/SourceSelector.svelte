<script lang="ts">
  import { viewModel, graphStore } from '../lib/stores/graphStore';
  import { groupSourceOptions } from '../lib/viewer/viewerState';

  function handleChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    graphStore.onSourceChange(target.value);
  }

  // Options with a `group` label (CLR-99: past-run sessions, grouped by
  // which watch run produced them) render inside an <optgroup>; the
  // live/frozen/in-memory-collection options stay top-level, unchanged
  // from before grouping existed.
  $: blocks = groupSourceOptions($viewModel.sourceOptions);
</script>

<select
  id="snapshot-source"
  class="bg-input text-foreground border-0 rounded px-2.5 py-1.5 text-xs font-medium focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all cursor-pointer hover:bg-input/80"
  value={$viewModel.sourceValue}
  onchange={handleChange}
>
  {#each blocks as block}
    {#if block.group}
      <optgroup label={block.group}>
        {#each block.options as option}
          <option value={option.value}>{option.text}</option>
        {/each}
      </optgroup>
    {:else}
      {#each block.options as option}
        <option value={option.value}>{option.text}</option>
      {/each}
    {/if}
  {/each}
</select>
