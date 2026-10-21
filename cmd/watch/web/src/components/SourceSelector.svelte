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
  class="h-9 cursor-pointer rounded-[10px] bg-muted px-3 text-body font-medium text-foreground outline-none transition-colors hover:bg-tab-active focus-visible:outline-2 focus-visible:outline-ring"
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
