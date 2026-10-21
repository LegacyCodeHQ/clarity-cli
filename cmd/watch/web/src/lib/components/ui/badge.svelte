<script lang="ts">
  import { cn } from "$lib/utils";
  import type { Snippet } from "svelte";

  interface Props {
    // "default" is filled and asks for attention. "outline" is quiet.
    variant?: "default" | "outline";
    class?: string;
    children?: Snippet;
  }

  let { variant = "default", class: className, children }: Props = $props();

  const variants = {
    default: "border-transparent bg-primary text-primary-foreground",
    outline: "border-input text-muted-foreground",
  };
</script>

<span
  class={cn(
    "inline-flex h-5 min-w-5 items-center justify-center rounded-md border px-1.5 text-caption leading-none font-semibold tabular-nums",
    variants[variant],
    className
  )}
>
  {@render children?.()}
</span>
