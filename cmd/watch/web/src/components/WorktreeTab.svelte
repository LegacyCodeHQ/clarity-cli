<script lang="ts">
  import GitBranch from "lucide-svelte/icons/git-branch";
  import FolderGit2 from "lucide-svelte/icons/folder-git-2";
  import FolderX from "lucide-svelte/icons/folder-x";
  import X from "lucide-svelte/icons/x";
  import { cn } from "$lib/utils";
  import Badge from "$lib/components/ui/badge.svelte";
  import NewDot from "$lib/components/ui/new-dot.svelte";
  import Tooltip from "$lib/components/ui/tooltip.svelte";
  import type { WorktreeDescriptor } from "$lib/protocol/viewerProtocol";
  import type { WorktreeChangeStatus } from "$lib/viewer/viewerState";

  interface Props {
    worktree: WorktreeDescriptor;
    selected: boolean;
    status: WorktreeChangeStatus;
    // Cap the tab's width so a long label truncates. Unset keeps the natural width.
    maxWidth?: number;
    onselect: () => void;
    onclose: () => void;
  }

  let { worktree, selected, status, maxWidth, onselect, onclose }: Props = $props();

  const label = $derived(worktree.label || worktree.id);
  const removed = $derived(!worktree.active);
  const Icon = $derived(removed ? FolderX : worktree.kind === "main" ? GitBranch : FolderGit2);
  const hasBadge = $derived(status !== "none");
  const trailing = $derived(hasBadge || removed);

  const badgeText = $derived.by(() => {
    const n = worktree.changedFiles;
    const files = `${n} ${n === 1 ? "file" : "files"} changed`;
    return status === "unseen" ? `${files}. New since you last looked.` : `${files}.`;
  });
</script>

{#snippet tab()}
  <div
    role="presentation"
    style:max-width={maxWidth ? `${maxWidth}px` : undefined}
    class={cn(
      "relative flex h-8 items-center rounded-lg text-xs font-medium transition-colors",
      "has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-ring",
      selected
        ? "bg-tab-active text-foreground shadow-xs ring-1 ring-tab-active-border ring-inset"
        : "hover:bg-tab-hover hover:text-foreground active:bg-tab-active active:text-foreground",
      !selected && !removed && "text-foreground/70",
      removed && "text-destructive hover:text-destructive"
    )}
  >
    <!-- The dot lives in the tab's left padding: 8px from the edge and 8px from
         the icon. The padding is always there, so the dot appearing or going
         moves nothing. -->
    {#if status === "unseen"}
      <NewDot class="absolute top-1/2 left-2 -translate-y-1/2" />
    {/if}
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      class={cn(
        "flex h-full min-w-0 flex-1 items-center pl-5.5 outline-none",
        trailing ? "pr-2" : "pr-3",
        removed && "italic"
      )}
      onclick={onselect}
    >
      <Icon size={16} class="mr-2 shrink-0" />
      <span class="truncate">{label}</span>
      {#if hasBadge}
        <Tooltip text={badgeText} class="ml-2 shrink-0">
          <Badge>{worktree.changedFiles}</Badge>
        </Tooltip>
      {/if}
    </button>
    {#if removed}
      <button
        type="button"
        aria-label={`Close ${label}`}
        class="mr-2 flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground outline-none hover:bg-border hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring"
        onclick={onclose}
      >
        <X size={12} />
      </button>
    {/if}
  </div>
{/snippet}

{#if removed}
  <Tooltip text="Worktree removed. History kept until you close it." class="min-w-0">
    {@render tab()}
  </Tooltip>
{:else}
  {@render tab()}
{/if}
