<script lang="ts">
  import Search from "lucide-svelte/icons/search";
  import X from "lucide-svelte/icons/x";
  import { cn } from "$lib/utils";

  interface Props {
    value?: string;
    placeholder?: string;
    class?: string;
    // The input element, so a parent can move focus to it.
    input?: HTMLInputElement | null;
    onkeydown?: (event: KeyboardEvent) => void;
  }

  let {
    value = $bindable(""),
    placeholder = "Search…",
    class: className,
    input = $bindable(null),
    onkeydown,
  }: Props = $props();

  function clear() {
    value = "";
    input?.focus();
  }
</script>

<div
  class={cn(
    "flex h-8 items-center gap-2 rounded-md border border-input px-2 text-body transition-colors",
    "hover:border-muted-foreground focus-within:border-ring focus-within:ring-1 focus-within:ring-ring",
    className
  )}
>
  <Search size={16} class="shrink-0 text-muted-foreground" />
  <input
    bind:this={input}
    bind:value
    {placeholder}
    {onkeydown}
    class="min-w-0 flex-1 bg-transparent text-foreground outline-none placeholder:text-muted-foreground"
  />
  {#if value}
    <button
      type="button"
      aria-label="Clear search"
      class="flex size-5 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground outline-none hover:text-foreground focus-visible:outline-2 focus-visible:outline-ring"
      onclick={clear}
    >
      <X size={12} />
    </button>
  {/if}
</div>
