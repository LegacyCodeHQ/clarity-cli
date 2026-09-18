package git

import "strings"

// RemoteOriginURL returns the "origin" remote's URL for the repo at path.
// It returns "" (with no error) when no such remote is configured — a fresh
// `git init` with nothing pushed anywhere is a normal, common state, not a
// failure. Callers that need a stable identity for a remote-less repo must
// decide their own fallback.
func RemoteOriginURL(path string) (string, error) {
	stdout, stderr, err := runGitCommand(path, "remote", "get-url", "origin")
	if err != nil {
		if strings.Contains(stderr, "No such remote") {
			return "", nil
		}
		return "", gitCommandError(err, stderr)
	}
	return strings.TrimSpace(string(stdout)), nil
}
