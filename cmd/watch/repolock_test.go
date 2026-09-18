package watch

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireRepoLock_SecondAttemptFindsItHeld(t *testing.T) {
	repo := initRepoWithCommit(t)

	first, existing, err := AcquireRepoLock(repo)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Nil(t, existing)
	defer first.Release() //nolint:errcheck

	second, existing, err := AcquireRepoLock(repo)
	require.NoError(t, err)
	assert.Nil(t, second, "a second process must not also acquire the lock")
	require.NotNil(t, existing)
	assert.Zero(t, existing.Port, "the holder hasn't called SetServing yet")
}

func TestRepoLock_SetServing_VisibleToALoser(t *testing.T) {
	repo := initRepoWithCommit(t)

	holder, _, err := AcquireRepoLock(repo)
	require.NoError(t, err)
	require.NotNil(t, holder)
	defer holder.Release() //nolint:errcheck

	require.NoError(t, holder.SetServing(4901))

	loser, existing, err := AcquireRepoLock(repo)
	require.NoError(t, err)
	assert.Nil(t, loser)
	require.NotNil(t, existing)
	assert.Equal(t, 4901, existing.Port)
	assert.Equal(t, os.Getpid(), existing.PID)
}

func TestRepoLock_Release_AllowsReacquire(t *testing.T) {
	repo := initRepoWithCommit(t)

	first, _, err := AcquireRepoLock(repo)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NoError(t, first.Release())

	second, existing, err := AcquireRepoLock(repo)
	require.NoError(t, err)
	require.NotNil(t, second, "the lock must be reacquirable once released")
	assert.Nil(t, existing)
	defer second.Release() //nolint:errcheck
}
