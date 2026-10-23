<script lang="ts">
  import GitBranch from "lucide-svelte/icons/git-branch";
  import FolderGit2 from "lucide-svelte/icons/folder-git-2";
  import FolderX from "lucide-svelte/icons/folder-x";
  import ChevronDown from "lucide-svelte/icons/chevron-down";
  import X from "lucide-svelte/icons/x";
  import { cn } from "$lib/utils";
  import Badge from "$lib/components/ui/badge.svelte";
  import NewDot from "$lib/components/ui/new-dot.svelte";
  import SearchField from "$lib/components/ui/search-field.svelte";
  import type { WorktreeDescriptor } from "$lib/protocol/viewerProtocol";
  import type { WorktreeChangeStatus } from "$lib/viewer/viewerState";

  interface Props {
    // The worktrees that did not fit in the bar, in tab order.
    worktrees: WorktreeDescriptor[];
    statusFor: (worktreeID: string) => WorktreeChangeStatus;
    onselect: (worktreeID: string) => void;
    onclose: (worktreeID: string) => void;
  }

  // The search field only earns its space once the list gets long.
  const SEARCH_THRESHOLD = 6;

  let { worktrees, statusFor, onselect, onclose }: Props = $props();

  let open = $state(false);
  let query = $state("");
  let highlighted = $state(-1);
  let root = $state<HTMLElement | null>(null);
  let trigger = $state<HTMLButtonElement | null>(null);
  let menu = $state<HTMLElement | null>(null);
  let searchInput = $state<HTMLInputElement | null>(null);
  let listEl = $state<HTMLElement | null>(null);
  // A long list fades out at the bottom edge while there is more to scroll to.
  let fadeBottom = $state(false);

  const showSearch = $derived(worktrees.length > SEARCH_THRESHOLD);
  const matches = $derived(
    worktrees.filter((w) => (w.label || w.id).toLowerCase().includes(query.trim().toLowerCase()))
  );
  const more = $derived(matches.filter((w) => w.active));
  const removed = $derived(matches.filter((w) => !w.active));
  const flat = $derived([...more, ...removed]);
  const hasUnseen = $derived(worktrees.some((w) => statusFor(w.id) === "unseen"));

  function iconFor(worktree: WorktreeDescriptor) {
    if (!worktree.active) return FolderX;
    return worktree.kind === "main" ? GitBranch : FolderGit2;
  }

  function toggle() {
    open = !open;
    if (open) {
      query = "";
      highlighted = -1;
      queueMicrotask(() => (searchInput ?? menu)?.focus());
    }
  }

  function close(refocus = false) {
    open = false;
    if (refocus) trigger?.focus();
  }

  function choose(worktreeID: string) {
    onselect(worktreeID);
    close();
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      event.preventDefault();
      close(true);
    } else if (event.key === "ArrowDown" && flat.length > 0) {
      event.preventDefault();
      highlighted = (highlighted + 1) % flat.length;
    } else if (event.key === "ArrowUp" && flat.length > 0) {
      event.preventDefault();
      highlighted = highlighted <= 0 ? flat.length - 1 : highlighted - 1;
    } else if (event.key === "Enter") {
      const target = flat[highlighted];
      if (target) {
        event.preventDefault();
        choose(target.id);
      }
    }
  }

  function handleWindowPointerDown(event: PointerEvent) {
    if (open && root && !root.contains(event.target as Node)) {
      close();
    }
  }

  function updateFade() {
    fadeBottom = !!listEl && listEl.scrollHeight - listEl.scrollTop - listEl.clientHeight > 1;
  }

  $effect(() => {
    open;
    flat;
    queueMicrotask(updateFade);
  });

  // Keep the keyboard highlight in view when the list scrolls.
  $effect(() => {
    if (highlighted >= 0) {
      menu?.querySelector(`[data-index="${highlighted}"]`)?.scrollIntoView({ block: "nearest" });
    }
  });

  // A search that changes the list invalidates the highlighted row.
  $effect(() => {
    query;
    highlighted = -1;
  });
</script>

<svelte:window onpointerdown={handleWindowPointerDown} />

{#snippet row(worktree: WorktreeDescriptor, index: number)}
  {@const Icon = iconFor(worktree)}
  {@const status = statusFor(worktree.id)}
  {@const isRemoved = !worktree.active}
  <div
    role="presentation"
    data-index={index}
    class={cn(
      "flex h-8 items-center rounded-md text-xs font-medium",
      index === highlighted && "bg-accent",
      "active:bg-pressed",
      isRemoved ? "text-destructive" : "text-foreground"
    )}
    onmouseenter={() => (highlighted = index)}
  >
    <button
      type="button"
      role="menuitem"
      class={cn("flex h-full min-w-0 flex-1 items-center gap-2 px-2 outline-none", isRemoved && "italic")}
      onclick={() => choose(worktree.id)}
    >
      <!-- Same convention as the tabs: the unseen dot sits on the icon's corner and
           the icon is cut away around it, so it works on any row background. -->
      <span class="relative flex size-4 shrink-0">
        <Icon
          size={16}
          class={cn(
            !isRemoved && "text-foreground/70",
            status === "unseen" && "[mask-image:radial-gradient(circle_at_13px_3px,transparent_5px,#000_5.5px)]"
          )}
        />
        {#if status === "unseen"}
          <NewDot class="absolute top-0 right-0" />
        {/if}
      </span>
      <span class="truncate">{worktree.label || worktree.id}</span>
      <span class="flex-1"></span>
      {#if status !== "none"}
        <Badge>{worktree.changedFiles}</Badge>
      {/if}
    </button>
    {#if isRemoved}
      <button
        type="button"
        aria-label={`Close ${worktree.label || worktree.id}`}
        class="mr-2 flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground outline-none hover:bg-border hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring"
        onclick={() => onclose(worktree.id)}
      >
        <X size={12} />
      </button>
    {/if}
  </div>
{/snippet}

<div bind:this={root} class="relative">
  <button
    bind:this={trigger}
    type="button"
    aria-haspopup="menu"
    aria-expanded={open}
    class={cn(
      "flex h-8 items-center gap-2 rounded-lg px-3 text-xs font-medium transition-colors outline-none",
      "focus-visible:outline-2 focus-visible:outline-ring",
      open
        ? "bg-tab-active text-foreground shadow-xs ring-1 ring-tab-active-border ring-inset"
        : "text-foreground/70 hover:bg-tab-hover hover:text-foreground active:bg-tab-active"
    )}
    onclick={toggle}
  >
    <span>+{worktrees.length}</span>
    {#if hasUnseen}
      <NewDot />
    {/if}
    <ChevronDown size={16} />
  </button>

  {#if open}
    <div
      bind:this={menu}
      role="menu"
      tabindex="-1"
      class="absolute top-full left-0 z-50 mt-2 w-80 rounded-[10px] border border-border bg-card p-1 shadow-lg outline-none"
      onkeydown={handleKeydown}
    >
      {#if showSearch}
        <SearchField bind:value={query} bind:input={searchInput} placeholder="Find worktree…" class="mb-2" />
      {/if}
      <div
        bind:this={listEl}
        onscroll={updateFade}
        class={cn(
          "max-h-60 overflow-y-auto",
          fadeBottom && "[mask-image:linear-gradient(to_bottom,#000_calc(100%-40px),transparent)]"
        )}
      >
        {#if flat.length === 0}
          <p class="flex h-12 items-center justify-center text-xs text-muted-foreground">
            No worktrees match “{query.trim()}”
          </p>
        {/if}
        {#if more.length > 0}
          <p class="px-2 pt-2 pb-1 text-caption font-semibold text-muted-foreground">More worktrees · {more.length}</p>
          {#each more as worktree, i (worktree.id)}
            {@render row(worktree, i)}
          {/each}
        {/if}
        {#if removed.length > 0}
          <p class={cn("px-2 pt-2 pb-1 text-caption font-semibold text-muted-foreground", more.length > 0 && "mt-4")}>
            Removed · {removed.length}
          </p>
          {#each removed as worktree, i (worktree.id)}
            {@render row(worktree, more.length + i)}
          {/each}
        {/if}
      </div>
    </div>
  {/if}
</div>
