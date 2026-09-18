<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Header from './components/Header.svelte';
  import WorktreeTabs from './components/WorktreeTabs.svelte';
  import GraphContainer from './components/GraphContainer.svelte';
  import Timeline from './components/Timeline.svelte';
  import { graphStore } from './lib/stores/graphStore';
  import { normalizeGraphStreamPayload, normalizePersistedSessionList } from './lib/protocol/viewerProtocol';

  interface Props {
    pageTitle: string;
  }

  let { pageTitle }: Props = $props();

  let connected = $state(false);
  let eventSource: EventSource | null = null;

  function connectSSE() {
    eventSource = new EventSource('/events');

    eventSource.addEventListener('graph', (event) => {
      try {
        const payload = normalizeGraphStreamPayload(JSON.parse(event.data));
        graphStore.mergePayload(payload);
      } catch (err) {
        console.error('Invalid graph payload:', err);
      }
    });

    eventSource.addEventListener('open', () => {
      connected = true;
    });

    eventSource.addEventListener('error', () => {
      connected = false;
    });
  }

  // Fetches the eager, metadata-only persisted-session listing (CLR-98's
  // GET /sessions) once on attach — cheap (no snapshot content), so unlike
  // a session's actual detail this isn't deferred to a user click. A 404
  // means persistence is disabled for this process; that's a normal,
  // silent case, not an error to surface.
  async function fetchPersistedSessions() {
    try {
      const res = await fetch('/sessions');
      if (!res.ok) {
        return;
      }
      graphStore.setPersistedSessions(normalizePersistedSessionList(await res.json()));
    } catch (err) {
      console.error('Failed to fetch persisted sessions:', err);
    }
  }

  onMount(() => {
    connectSSE();
    fetchPersistedSessions();
  });

  onDestroy(() => {
    if (eventSource) {
      eventSource.close();
    }
  });
</script>

<div class="h-screen flex flex-col bg-background">
  <Header {pageTitle} {connected} />
  <WorktreeTabs />
  <GraphContainer />
  <Timeline />
</div>
