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
    "inline-flex h-5 min-w-5 items-center justify-center rounded-md border-[1.5px] border-foreground/35 px-1.5 text-caption leading-none font-semibold tabular-nums text-foreground/70",
    className
  )}
>
  {@render children?.()}
</span>
