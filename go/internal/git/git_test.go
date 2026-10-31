package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wade/aia/agent-runtime/internal/git"
)

func TestGitOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agent-git-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	client := git.NewClient(tmpDir)

	// 1. Init
	if err := client.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if !client.IsRepo() {
		t.Fatalf("expected IsRepo() to be true")
	}

	// 2. Initial commit
	testFile := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	hasUncommitted, err := client.HasUncommittedChanges()
	if err != nil || !hasUncommitted {
		t.Fatalf("expected uncommitted changes before commit")
	}

	if err := client.AddAndCommit("initial commit"); err != nil {
		t.Fatalf("AddAndCommit failed: %v", err)
	}

	baseCommit, err := client.GetCurrentCommit()
	if err != nil || baseCommit == "" {
		t.Fatalf("GetCurrentCommit failed: %v", err)
	}

	// 3. Modify file
	if err := os.WriteFile(testFile, []byte("hello world modified\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	diff, err := client.GetDiff(baseCommit)
	if err != nil || diff == "" {
		t.Fatalf("expected diff against baseCommit, got: %q, err: %v", diff, err)
	}

	changedFiles, err := client.GetChangedFiles(baseCommit)
	if err != nil || len(changedFiles) != 1 || changedFiles[0] != "hello.txt" {
		t.Fatalf("unexpected changed files: %v, err: %v", changedFiles, err)
	}
}
