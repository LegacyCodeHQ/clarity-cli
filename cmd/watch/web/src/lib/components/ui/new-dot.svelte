<script lang="ts">
  import { cn } from "$lib/utils";

  interface Props {
    class?: string;
  }

  let { class: className }: Props = $props();
</script>

<!-- Marks something as new since the user last looked. Says nothing about how much. -->
<span role="img" aria-label="New since you last looked" class={cn("size-1.5 shrink-0 rounded-full bg-primary", className)}></span>
