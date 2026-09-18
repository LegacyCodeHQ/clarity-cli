package store

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
)

func seedWorktree(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	projectID := seedProject(t, db)
	require.NoError(t, RegisterWorktree(db, id, projectID, "/repo", protocol.WorktreeKindMain, "label"))
}

func TestOpenSession_FirstCall_CreatesSessionNumberOne(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	id, err := OpenSession(db, "main")
	require.NoError(t, err)
	assert.NotZero(t, id)

	var worktreeID string
	var number int
	var closedAt sql.NullTime
	require.NoError(t, db.QueryRow(
		`SELECT worktree_id, number, closed_at FROM sessions WHERE id = ?`, id,
	).Scan(&worktreeID, &number, &closedAt))
	assert.Equal(t, "main", worktreeID)
	assert.Equal(t, 1, number)
	assert.False(t, closedAt.Valid, "a freshly opened session must not be closed")
}

func TestOpenSession_RepeatedCalls_AlwaysCreateNewSessionsIncrementingNumber(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")

	first, err := OpenSession(db, "main")
	require.NoError(t, err)
	second, err := OpenSession(db, "main")
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "OpenSession never resumes a prior session")

	var firstNumber, secondNumber int
	require.NoError(t, db.QueryRow(`SELECT number FROM sessions WHERE id = ?`, first).Scan(&firstNumber))
	require.NoError(t, db.QueryRow(`SELECT number FROM sessions WHERE id = ?`, second).Scan(&secondNumber))
	assert.Equal(t, firstNumber+1, secondNumber)
}

func TestOpenSession_DifferentWorktrees_IndependentNumbering(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	seedWorktree(t, db, "wt-a")

	mainID, err := OpenSession(db, "main")
	require.NoError(t, err)
	otherID, err := OpenSession(db, "wt-a")
	require.NoError(t, err)

	var mainNumber, otherNumber int
	require.NoError(t, db.QueryRow(`SELECT number FROM sessions WHERE id = ?`, mainID).Scan(&mainNumber))
	require.NoError(t, db.QueryRow(`SELECT number FROM sessions WHERE id = ?`, otherID).Scan(&otherNumber))
	assert.Equal(t, 1, mainNumber)
	assert.Equal(t, 1, otherNumber, "each worktree has its own independent numbering sequence")
}

func TestCloseSessionCommitted_SetsClosedReasonAndInsertsCommits(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	sessionID, err := OpenSession(db, "main")
	require.NoError(t, err)

	commits := []CommitRecord{
		{Position: 0, Hash: "aaa111", Subject: "first"},
		{Position: 1, Hash: "bbb222", Subject: "second"},
	}
	require.NoError(t, CloseSessionCommitted(db, sessionID, commits))

	var closedAt sql.NullTime
	var closedReason string
	require.NoError(t, db.QueryRow(
		`SELECT closed_at, closed_reason FROM sessions WHERE id = ?`, sessionID,
	).Scan(&closedAt, &closedReason))
	assert.True(t, closedAt.Valid)
	assert.Equal(t, "committed", closedReason)

	rows, err := db.Query(`SELECT position, hash, subject FROM commits WHERE session_id = ? ORDER BY position`, sessionID)
	require.NoError(t, err)
	defer rows.Close()
	var got []CommitRecord
	for rows.Next() {
		var c CommitRecord
		require.NoError(t, rows.Scan(&c.Position, &c.Hash, &c.Subject))
		got = append(got, c)
	}
	assert.Equal(t, commits, got)
}

func TestCloseSessionCommitted_NoCommits_Errors(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	sessionID, err := OpenSession(db, "main")
	require.NoError(t, err)

	err = CloseSessionCommitted(db, sessionID, nil)
	assert.Error(t, err)

	var closedAt sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT closed_at FROM sessions WHERE id = ?`, sessionID).Scan(&closedAt))
	assert.False(t, closedAt.Valid, "a rejected close must not leave the session half-closed")
}

func TestCloseSessionAbandoned_SetsClosedReasonNoCommits(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	sessionID, err := OpenSession(db, "main")
	require.NoError(t, err)

	require.NoError(t, CloseSessionAbandoned(db, sessionID))

	var closedAt sql.NullTime
	var closedReason string
	require.NoError(t, db.QueryRow(
		`SELECT closed_at, closed_reason FROM sessions WHERE id = ?`, sessionID,
	).Scan(&closedAt, &closedReason))
	assert.True(t, closedAt.Valid)
	assert.Equal(t, "abandoned", closedReason)

	var commitCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM commits WHERE session_id = ?`, sessionID).Scan(&commitCount))
	assert.Zero(t, commitCount)
}

func TestCloseSessionWorktreeRemoved_SetsClosedReasonNoCommits(t *testing.T) {
	db := openMigratedTestDB(t)
	seedWorktree(t, db, "main")
	sessionID, err := OpenSession(db, "main")
	require.NoError(t, err)

	require.NoError(t, CloseSessionWorktreeRemoved(db, sessionID))

	var closedReason string
	require.NoError(t, db.QueryRow(`SELECT closed_reason FROM sessions WHERE id = ?`, sessionID).Scan(&closedReason))
	assert.Equal(t, "worktree_removed", closedReason)
}
