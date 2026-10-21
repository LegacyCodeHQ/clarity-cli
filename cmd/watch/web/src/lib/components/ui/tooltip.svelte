<script lang="ts">
  import { cn } from "$lib/utils";
  import type { Snippet } from "svelte";

  interface Props {
    text: string;
    class?: string;
    children: Snippet;
  }

  let { text, class: className, children }: Props = $props();
</script>

<span class={cn("group/tooltip relative inline-flex", className)}>
  {@render children()}
  <span
    role="tooltip"
    class="pointer-events-none invisible absolute top-full left-1/2 z-50 mt-1 -translate-x-1/2 whitespace-nowrap rounded-md bg-foreground px-3 py-1.5 text-xs font-normal not-italic text-background opacity-0 transition-opacity group-hover/tooltip:visible group-hover/tooltip:opacity-100 group-hover/tooltip:delay-500 group-focus-within/tooltip:visible group-focus-within/tooltip:opacity-100"
  >
    {text}
  </span>
</span>
