package watch

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofrs/flock"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/store"
)

// RepoLock is the exclusive advisory lock that gates persistence and serving
// to a single clarity watch process per repository (see CLR-91). It is held
// for the lifetime of the process; the OS releases it automatically on exit,
// including a crash, so there is no stale-lock/PID-liveness bookkeeping to
// build or recover.
type RepoLock struct {
	fl *flock.Flock
}

// lockInfo is the JSON written into the lock file once its holder is
// actually serving, so a process that loses the race can tell the user
// where the existing instance is instead of silently starting a second one.
// The zero value means "held, but the holder hasn't recorded its port yet"
// — the brief startup window between acquiring the lock and binding a port,
// not an error.
type lockInfo struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
}

// AcquireRepoLock tries to become the sole clarity watch process for the
// repository containing worktreePath. On success it returns the held lock,
// ready for SetServing once the port is known, and a nil lockInfo. If
// another process already holds it, it returns a nil *RepoLock and that
// process's last recorded lockInfo instead.
func AcquireRepoLock(worktreePath string) (*RepoLock, *lockInfo, error) {
	path, err := store.LockPathFor(worktreePath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve repo lock path: %w", err)
	}

	fl := flock.New(path)
	locked, err := fl.TryLock()
	if err != nil {
		return nil, nil, fmt.Errorf("acquire repo lock: %w", err)
	}
	if !locked {
		info := readLockInfo(path)
		return nil, &info, nil
	}
	return &RepoLock{fl: fl}, nil, nil
}

func readLockInfo(path string) lockInfo {
	var info lockInfo
	data, err := os.ReadFile(path)
	if err != nil {
		return info
	}
	_ = json.Unmarshal(data, &info) // zero-valued on any parse failure (e.g. holder hasn't written yet)
	return info
}

// SetServing records the port this process is serving on, so a process that
// loses the lock race can point the user at it instead of starting a
// competing instance.
func (l *RepoLock) SetServing(port int) error {
	data, err := json.Marshal(lockInfo{Port: port, PID: os.Getpid()})
	if err != nil {
		return fmt.Errorf("encode repo lock info: %w", err)
	}
	if err := os.WriteFile(l.fl.Path(), data, 0o644); err != nil {
		return fmt.Errorf("write repo lock info: %w", err)
	}
	return nil
}

// Release releases the lock. The file itself is left in place — its
// content goes stale but is harmless, and the next holder overwrites it via
// SetServing.
func (l *RepoLock) Release() error {
	return l.fl.Unlock()
}
