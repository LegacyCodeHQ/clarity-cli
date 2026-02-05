package graph

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGraphInputDirectory_WithUnsupportedFiles_RendersStandaloneNodes(t *testing.T) {
	repoDir := t.TempDir()
	dir := filepath.Join(repoDir, "src")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}

	unsupportedFile := filepath.Join(dir, "Main.java")
	if err := os.WriteFile(unsupportedFile, []byte("class Main {}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cmd := NewCommand()
	cmd.SetArgs([]string{"-i", dir, "-f", "dot"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"Main.java"`) {
		t.Fatalf("expected graph output to include Main.java node, got:\n%s", output)
	}
	if strings.Contains(output, "->") {
		t.Fatalf("expected no dependency edges for unsupported-only input, got:\n%s", output)
	}
}

func TestGraphCommit_WithUnsupportedFiles_RendersStandaloneNodes(t *testing.T) {
	repoDir := t.TempDir()
	gitInitRepo(t, repoDir)

	unsupportedFile := filepath.Join(repoDir, "model", "Main.java")
	if err := os.MkdirAll(filepath.Dir(unsupportedFile), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(unsupportedFile, []byte("class Main {}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	gitRun(t, repoDir, "add", ".")
	gitRun(t, repoDir, "commit", "-m", "add java file")

	cmd := NewCommand()
	cmd.SetArgs([]string{"-r", repoDir, "-c", "HEAD", "-f", "dot"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `"Main.java"`) {
		t.Fatalf("expected graph output to include Main.java node, got:\n%s", output)
	}
	if strings.Contains(output, "->") {
		t.Fatalf("expected no dependency edges for unsupported-only commit, got:\n%s", output)
	}
}

func TestGraphInput_WithSupportedFiles_RendersNode(t *testing.T) {
	repoDir := t.TempDir()
	supportedFile := filepath.Join(repoDir, "main.go")
	if err := os.WriteFile(supportedFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cmd := NewCommand()
	cmd.SetArgs([]string{"-i", supportedFile, "-f", "dot"})

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"main.go"`) {
		t.Fatalf("expected graph output to include main.go node, got:\n%s", stdout.String())
	}
}

func gitInitRepo(t *testing.T, repoDir string) {
	t.Helper()

	gitRun(t, repoDir, "init")
	gitRun(t, repoDir, "config", "user.name", "test")
	gitRun(t, repoDir, "config", "user.email", "test@example.com")
}

func gitRun(t *testing.T, repoDir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v failed: %v\nstderr: %s", args, err, strings.TrimSpace(stderr.String()))
	}
}
