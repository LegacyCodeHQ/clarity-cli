<script lang="ts">
  import { cn } from "$lib/utils";
  import type { Snippet } from "svelte";

  interface Props {
    class?: string;
    children?: Snippet;
  }

  let { class: className, children }: Props = $props();
</script>

<span
  class={cn(
    "inline-flex h-4 min-w-4 items-center justify-center rounded-sm bg-count px-1 text-caption leading-none font-medium tabular-nums text-foreground/70",
    className
  )}
>
  {@render children?.()}
</span>
