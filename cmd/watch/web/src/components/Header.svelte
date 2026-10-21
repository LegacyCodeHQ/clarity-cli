<script lang="ts">
  import SourceSelector from './SourceSelector.svelte';
  import ConnectionIndicator from './ConnectionIndicator.svelte';
  import WorktreeTabs from './WorktreeTabs.svelte';

  interface Props {
    pageTitle: string;
    connected: boolean;
  }

  let { pageTitle, connected }: Props = $props();

  // The page title is "<project> • clarity watch"; the bar only wants the project.
  const project = $derived(pageTitle.split(' • ')[0] || pageTitle);
</script>

<header class="chrome flex h-12 items-center gap-4 border-b border-border bg-background px-4">
  <div class="flex shrink-0 items-center gap-2">
    <h1 class="font-code text-sm font-semibold text-foreground">{project}</h1>
    <ConnectionIndicator {connected} />
  </div>
  <WorktreeTabs />
  <SourceSelector />
</header>
