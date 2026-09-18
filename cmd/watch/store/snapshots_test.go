package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendSnapshot_InsertsRow(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	runID := seedRun(t, db)
	sessionID, err := OpenSession(db, "main", runID)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, AppendSnapshot(db, sessionID, 0, "digraph{}", "dot", "baseline", now))

	var source, format, kind string
	var position int
	require.NoError(t, db.QueryRow(
		`SELECT position, source, format, kind FROM snapshots WHERE session_id = ?`, sessionID,
	).Scan(&position, &source, &format, &kind))
	assert.Zero(t, position)
	assert.Equal(t, "digraph{}", source)
	assert.Equal(t, "dot", format)
	assert.Equal(t, "baseline", kind)
}

func TestAppendSnapshot_MultipleCalls_EachInsertsIndependently(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	runID := seedRun(t, db)
	sessionID, err := OpenSession(db, "main", runID)
	require.NoError(t, err)

	now := time.Now().UTC()
	require.NoError(t, AppendSnapshot(db, sessionID, 0, "digraph{a}", "dot", "baseline", now))
	require.NoError(t, AppendSnapshot(db, sessionID, 1, "digraph{a;b}", "dot", "incremental", now))

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM snapshots WHERE session_id = ?`, sessionID).Scan(&count))
	assert.Equal(t, 2, count, "each publish must persist immediately, not batch")
}
