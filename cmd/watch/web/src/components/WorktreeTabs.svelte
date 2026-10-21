<script lang="ts">
  import { onMount } from "svelte";
  import { graphStore, viewModel } from "../lib/stores/graphStore";
  import { getWorktreeChangeStatus } from "../lib/viewer/viewerState";
  import { computeTabLayout } from "../lib/viewer/tabLayout";
  import WorktreeTab from "./WorktreeTab.svelte";
  import WorktreeOverflowMenu from "./WorktreeOverflowMenu.svelte";

  // Width reserved for the "+N" button when some tabs move into its menu.
  const OVERFLOW_WIDTH = 88;
  // The tab list's own padding on both sides (p-0.5) and the space between tabs (gap-1).
  const LIST_PADDING = 4;
  const TAB_GAP = 4;

  let host = $state<HTMLElement | null>(null);
  let measurer = $state<HTMLElement | null>(null);
  let available = $state(Infinity);
  let widths = $state<Record<string, number>>({});

  const worktrees = $derived($viewModel.state.worktrees);
  const selectedID = $derived($viewModel.state.selectedWorktreeID);
  // A lone worktree needs no tabs: the project name says everything.
  const showTabs = $derived(worktrees.length > 1 || worktrees.some((w) => !w.active));

  function statusFor(worktreeID: string) {
    return getWorktreeChangeStatus($viewModel.state, worktreeID);
  }

  const layout = $derived.by(() => {
    const measured = worktrees.every((w) => widths[w.id] !== undefined);
    if (!measured) {
      return { visible: worktrees.map((w) => ({ id: w.id, maxWidth: undefined as number | undefined })), overflow: [] as string[] };
    }
    return computeTabLayout(
      worktrees.map((w) => ({
        id: w.id,
        width: widths[w.id] ?? 0,
        isMain: w.kind === "main",
        selected: w.id === selectedID,
        active: w.active,
        status: statusFor(w.id),
      })),
      available - LIST_PADDING,
      { overflowWidth: OVERFLOW_WIDTH, gap: TAB_GAP }
    );
  });

  const overflowWorktrees = $derived(
    layout.overflow.map((id) => worktrees.find((w) => w.id === id)).filter((w) => w !== undefined)
  );

  // Anything that changes how wide a tab is.
  const signature = $derived(
    worktrees.map((w) => `${w.id}:${w.label}:${w.active}:${w.kind}:${w.changedFiles}:${statusFor(w.id)}`).join("|")
  );

  function measure() {
    if (!measurer) {
      return;
    }
    const next: Record<string, number> = {};
    for (const el of Array.from(measurer.children) as HTMLElement[]) {
      const id = el.dataset.id;
      if (id) {
        next[id] = Math.ceil(el.getBoundingClientRect().width);
      }
    }
    if (JSON.stringify(next) !== JSON.stringify(widths)) {
      widths = next;
    }
  }

  $effect(() => {
    signature;
    measure();
  });

  onMount(() => {
    const hostObserver = new ResizeObserver(() => {
      available = host?.clientWidth ?? Infinity;
    });
    // Web fonts arriving later change tab widths, so re-measure when the
    // hidden copy resizes.
    const measurerObserver = new ResizeObserver(measure);
    if (host) hostObserver.observe(host);
    if (measurer) measurerObserver.observe(measurer);
    document.fonts?.ready.then(measure);
    return () => {
      hostObserver.disconnect();
      measurerObserver.disconnect();
    };
  });

  function handleSelect(worktreeID: string) {
    graphStore.onSelectWorktree(worktreeID);
  }

  // Tear down a finished (inactive) worktree tab. The server drops the tab and
  // broadcasts the new set, so the UI updates through the normal SSE flow — no
  // local state mutation here. Active tabs never expose this control.
  async function handleClose(worktreeID: string) {
    try {
      await fetch(`/worktrees/${encodeURIComponent(worktreeID)}/close`, { method: "POST" });
    } catch (err) {
      console.error("Failed to close worktree tab:", err);
    }
  }
</script>

<div bind:this={host} class="relative flex min-w-0 flex-1 items-center">
  {#if showTabs}
    <div role="tablist" aria-label="Working trees" class="inline-flex h-9 items-center gap-1 rounded-[10px] bg-muted p-0.5">
      {#each layout.visible as slot (slot.id)}
        {@const worktree = worktrees.find((w) => w.id === slot.id)}
        {#if worktree}
          <WorktreeTab
            {worktree}
            selected={worktree.id === selectedID}
            status={statusFor(worktree.id)}
            maxWidth={slot.maxWidth}
            onselect={() => handleSelect(worktree.id)}
            onclose={() => handleClose(worktree.id)}
          />
        {/if}
      {/each}
      {#if overflowWorktrees.length > 0}
        <WorktreeOverflowMenu
          worktrees={overflowWorktrees}
          {statusFor}
          onselect={handleSelect}
          onclose={handleClose}
        />
      {/if}
    </div>
  {/if}

  <!-- A hidden copy of every tab at its natural width, so the layout rules
       know how much room each one needs. -->
  <div bind:this={measurer} aria-hidden="true" class="pointer-events-none invisible absolute top-0 left-0 flex w-max">
    {#each worktrees as worktree (worktree.id)}
      <div data-id={worktree.id} class="shrink-0">
        <WorktreeTab
          {worktree}
          selected={worktree.id === selectedID}
          status={statusFor(worktree.id)}
          onselect={() => {}}
          onclose={() => {}}
        />
      </div>
    {/each}
  </div>
</div>
