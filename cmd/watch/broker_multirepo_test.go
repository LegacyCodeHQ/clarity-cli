package watch

import (
	"testing"
	"time"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBroker_RegisterWorktree_EmitsTabsToSubscribers(t *testing.T) {
	b := newBroker()
	ch := b.subscribe()
	defer b.unsubscribe(ch)

	b.registerWorktree(protocol.WorktreeDescriptor{
		ID:    "primary",
		Path:  "/repo",
		Label: "clarity-cli",
		Kind:  protocol.WorktreeKindMain,
	})

	select {
	case got := <-ch:
		require.Len(t, got.Worktrees, 1)
		assert.Equal(t, "primary", got.Worktrees[0].ID)
		assert.Equal(t, protocol.WorktreeKindMain, got.Worktrees[0].Kind)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tab descriptor")
	}
}

func TestBroker_PublishToMultipleWorktrees_FlatPayloadTaggedByWorktreeID(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Label: "primary", Kind: protocol.WorktreeKindMain})
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "wt-aaaaaaaa", Path: "/tmp/wt", Label: "wt", Kind: protocol.WorktreeKindLinked})

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	b.publish("primary", "digraph primary { A; }")
	b.publish("wt-aaaaaaaa", "digraph wt { B; }")

	// Drain until we see both repos represented.
	deadline := time.After(2 * time.Second)
	var last protocol.GraphStreamPayload
	for {
		select {
		case got := <-ch:
			last = got
			if len(got.WorkingSnapshots) == 2 {
				goto done
			}
		case <-deadline:
			t.Fatalf("timed out; last payload had %d snapshots", len(last.WorkingSnapshots))
		}
	}
done:
	require.Len(t, last.WorkingSnapshots, 2)
	repoIDs := []string{last.WorkingSnapshots[0].WorktreeID, last.WorkingSnapshots[1].WorktreeID}
	assert.ElementsMatch(t, []string{"primary", "wt-aaaaaaaa"}, repoIDs)

	for _, s := range last.WorkingSnapshots {
		if s.WorktreeID == "primary" {
			assert.Equal(t, "digraph primary { A; }", s.DOT)
		} else {
			assert.Equal(t, "digraph wt { B; }", s.DOT)
		}
	}
}

func TestBroker_MarksSessionStartOncePerWorktree(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain})

	// The first snapshot recorded for a repo is the session start: it captures
	// whatever already existed in the working tree when the watcher attached.
	b.publish("primary", "digraph { A; }")
	// Subsequent live snapshots are not session starts.
	b.publish("primary", "digraph { A; B; }")

	ch := b.subscribe()
	got := <-ch
	require.Len(t, got.WorkingSnapshots, 2)
	assert.True(t, got.WorkingSnapshots[0].SessionStart, "first snapshot should be marked session start")
	assert.False(t, got.WorkingSnapshots[1].SessionStart, "second snapshot should not be a session start")
	b.unsubscribe(ch)

	// A commit archives the working set; the next snapshot belongs to a new
	// cycle but is still mid-session — it must NOT be a fresh session start.
	b.archiveWorkingSet("primary")
	b.publish("primary", "digraph { C; }")

	ch2 := b.subscribe()
	got2 := <-ch2
	defer b.unsubscribe(ch2)
	require.Len(t, got2.WorkingSnapshots, 1)
	assert.False(t, got2.WorkingSnapshots[0].SessionStart, "post-archive snapshot is mid-session, not a session start")
	// The archived collection keeps the original session-start marker so the
	// boundary stays visible when browsing past cycles.
	require.Len(t, got2.PastCollections, 1)
	require.NotEmpty(t, got2.PastCollections[0].Snapshots)
	assert.True(t, got2.PastCollections[0].Snapshots[0].SessionStart, "archived first snapshot retains session start")
}

func TestBroker_ArchiveOnlyAffectsThatWorktree(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain})
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "wt-aaaaaaaa", Path: "/tmp/wt", Kind: protocol.WorktreeKindLinked})

	b.publish("primary", "digraph p {}")
	b.publish("wt-aaaaaaaa", "digraph w {}")

	b.archiveWorkingSet("primary")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	select {
	case got := <-ch:
		// wt's working snapshot should remain; primary's was archived.
		require.Len(t, got.WorkingSnapshots, 1)
		assert.Equal(t, "wt-aaaaaaaa", got.WorkingSnapshots[0].WorktreeID)
		require.Len(t, got.PastCollections, 1)
		assert.Equal(t, "primary", got.PastCollections[0].WorktreeID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payload")
	}
}

func TestBroker_UnregisterWorktree_DropsTabAndHistory(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain})
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "wt-aaaaaaaa", Path: "/tmp/wt", Kind: protocol.WorktreeKindLinked})
	b.publish("primary", "digraph p {}")
	b.publish("wt-aaaaaaaa", "digraph w {}")

	b.unregisterWorktree("wt-aaaaaaaa")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	select {
	case got := <-ch:
		require.Len(t, got.Worktrees, 1)
		assert.Equal(t, "primary", got.Worktrees[0].ID)
		require.Len(t, got.WorkingSnapshots, 1)
		assert.Equal(t, "primary", got.WorkingSnapshots[0].WorktreeID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payload after unregister")
	}
}

// markWorktreeFinished is the removal path for a worktree whose git working tree
// was deleted: the tab must stay visible, but its final working snapshots are
// no longer live and should be exposed as an archived collection.
func TestBroker_MarkWorktreeFinished_ArchivesFinalWorkingSet(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "wt-aaaaaaaa", Path: "/tmp/wt", Kind: protocol.WorktreeKindLinked, Active: true})
	b.publish("primary", "digraph p {}")
	b.publish("wt-aaaaaaaa", "digraph w {}")

	b.markWorktreeFinished("wt-aaaaaaaa")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	select {
	case got := <-ch:
		// Both tabs remain; the removed worktree is now inactive.
		require.Len(t, got.Worktrees, 2)
		byID := make(map[string]protocol.WorktreeDescriptor)
		for _, r := range got.Worktrees {
			byID[r.ID] = r
		}
		assert.True(t, byID["primary"].Active, "primary worktree stays active")
		assert.False(t, byID["wt-aaaaaaaa"].Active, "removed worktree flips to inactive")
		require.Len(t, got.WorkingSnapshots, 1)
		assert.Equal(t, "primary", got.WorkingSnapshots[0].WorktreeID)
		require.Len(t, got.PastCollections, 1)
		assert.Equal(t, "wt-aaaaaaaa", got.PastCollections[0].WorktreeID)
		require.Len(t, got.PastCollections[0].Snapshots, 1)
		assert.Equal(t, "digraph w {}", got.PastCollections[0].Snapshots[0].DOT)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payload after markWorktreeFinished")
	}
}

// closeWorktree is the user-initiated teardown of a finished tab: only inactive
// worktrees can be closed, and closing drops the tab and its history.
func TestBroker_CloseWorktree_RemovesFinishedTab(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "wt-aaaaaaaa", Path: "/tmp/wt", Kind: protocol.WorktreeKindLinked, Active: true})
	b.publish("primary", "digraph p {}")
	b.publish("wt-aaaaaaaa", "digraph w {}")
	b.markWorktreeFinished("wt-aaaaaaaa")

	assert.Equal(t, closeOK, b.closeWorktree("wt-aaaaaaaa"), "closing a finished worktree should succeed")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	select {
	case got := <-ch:
		require.Len(t, got.Worktrees, 1)
		assert.Equal(t, "primary", got.Worktrees[0].ID)
		require.Len(t, got.WorkingSnapshots, 1)
		assert.Equal(t, "primary", got.WorkingSnapshots[0].WorktreeID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payload after closeWorktree")
	}
}

// An active worktree's tab is pinned: closeWorktree must refuse it so the UI can't
// tear down a tab that's still being watched.
func TestBroker_CloseWorktree_RefusesActiveWorktree(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})
	b.publish("primary", "digraph p {}")

	assert.Equal(t, closeActive, b.closeWorktree("primary"), "closing an active worktree should be refused")

	ch := b.subscribe()
	defer b.unsubscribe(ch)

	select {
	case got := <-ch:
		require.Len(t, got.Worktrees, 1, "active tab must remain after a refused close")
		assert.Equal(t, "primary", got.Worktrees[0].ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for payload")
	}
}

func TestBroker_CloseWorktree_UnknownWorktreeReturnsFalse(t *testing.T) {
	b := newBroker()
	b.registerWorktree(protocol.WorktreeDescriptor{ID: "primary", Path: "/repo", Kind: protocol.WorktreeKindMain, Active: true})

	assert.Equal(t, closeNotFound, b.closeWorktree("wt-missing"), "closing an unknown worktree should report not found")
}
