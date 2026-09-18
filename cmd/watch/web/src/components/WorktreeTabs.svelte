<script lang="ts">
  import { graphStore, viewModel } from '../lib/stores/graphStore';

  function handleClick(worktreeID: string) {
    graphStore.onSelectWorktree(worktreeID);
  }

  // Tear down a finished (inactive) worktree tab. The server drops the tab and
  // broadcasts the new set, so the UI updates through the normal SSE flow — no
  // local state mutation here. Active tabs never expose this control.
  async function handleClose(event: MouseEvent, worktreeID: string) {
    event.stopPropagation();
    try {
      await fetch(`/worktrees/${encodeURIComponent(worktreeID)}/close`, { method: 'POST' });
    } catch (err) {
      console.error('Failed to close worktree tab:', err);
    }
  }
</script>

{#if $viewModel.state.worktrees.length > 1 || $viewModel.state.worktrees.some((w) => !w.active)}
  <div
    class="px-4 pt-1.5 pb-0 bg-card border-b border-border flex items-end gap-1 overflow-x-auto"
    role="tablist"
    aria-label="Working trees"
  >
    {#each $viewModel.state.worktrees as worktree (worktree.id)}
      {@const isActive = worktree.id === $viewModel.state.selectedWorktreeID}
      <button
        type="button"
        role="tab"
        aria-selected={isActive}
        title={worktree.active ? worktree.path : `${worktree.path} (removed)`}
        class="group flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-t border-b-2 whitespace-nowrap transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary/40"
        class:border-primary={isActive}
        class:text-foreground={isActive && worktree.active}
        class:bg-background={isActive}
        class:border-transparent={!isActive}
        class:text-muted-foreground={!isActive || !worktree.active}
        class:italic={!worktree.active}
        class:hover:text-foreground={!isActive && worktree.active}
        class:hover:bg-input={!isActive}
        onclick={() => handleClick(worktree.id)}
      >
        <span class="truncate">{worktree.label || worktree.id}</span>
        {#if !worktree.active}
          <span
            role="button"
            tabindex="0"
            aria-label={`Close ${worktree.label || worktree.id}`}
            title="Close tab"
            class="flex items-center justify-center w-4 h-4 rounded-sm text-muted-foreground hover:text-foreground hover:bg-destructive/20 leading-none"
            onclick={(e) => handleClose(e, worktree.id)}
            onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { handleClose(e as unknown as MouseEvent, worktree.id); } }}
          >
            ×
          </span>
        {/if}
      </button>
    {/each}
  </div>
{/if}
