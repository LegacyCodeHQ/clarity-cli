package lean

import (
	"path/filepath"
	"sync"

	"github.com/LegacyCodeHQ/clarity/vcs"
)

// lakeLookup lives for one resolver/reader snapshot. Both successful and absent
// configuration lookups are shared by workers, never across graph analyses.
type lakeLookup struct {
	reader      vcs.ContentReader
	directories sync.Map
}

type lakeLookupEntry struct {
	once  sync.Once
	roots []string
}

func (l *lakeLookup) sourceRoots(absPath string) []string {
	roots := l.forDirectory(filepath.Dir(absPath))
	if roots == nil {
		return ancestorSourceRoots(absPath)
	}
	return roots
}

func (l *lakeLookup) forDirectory(dir string) []string {
	value, found := l.directories.Load(dir)
	if !found {
		value, _ = l.directories.LoadOrStore(dir, &lakeLookupEntry{})
	}
	entry := value.(*lakeLookupEntry)
	entry.once.Do(func() {
		if config, found := readLakePackage(dir, l.reader); found {
			entry.roots = lakeSourceRoots(dir, config)
		} else if parent := filepath.Dir(dir); parent != dir {
			entry.roots = l.forDirectory(parent)
		}
	})
	return entry.roots
}
