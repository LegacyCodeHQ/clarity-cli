package watch

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/LegacyCodeHQ/clarity/cmd/watch/protocol"
)

const mainWorktreeID = "main"

// worktreeIDFor returns the stable id used to key all snapshots/collections
// for a working tree. The main worktree (when watch was launched from
// inside one) gets the literal id "main"; every other tree gets
// "wt-<hash8>", an 8-character hex digest of its absolute path.
func worktreeIDFor(absPath string, kind protocol.WorktreeKind) string {
	if kind == protocol.WorktreeKindMain {
		return mainWorktreeID
	}
	sum := sha256.Sum256([]byte(absPath))
	return "wt-" + hex.EncodeToString(sum[:])[:8]
}

// primaryRepoLabel returns the tab label for the currently checked-out tree.
// The branch is more useful than the project directory here because the first
// tab is always the current checkout.
func primaryRepoLabel(absPath, branch string) string {
	short := strings.TrimPrefix(branch, "refs/heads/")
	if short != "" {
		return short
	}
	return linkedRepoLabel(absPath)
}

// linkedRepoLabel returns the tab label for an additional worktree.
func linkedRepoLabel(absPath string) string {
	base := filepath.Base(filepath.Clean(absPath))
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}
