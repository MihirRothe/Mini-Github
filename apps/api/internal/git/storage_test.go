package git

import (
	"os"
	"testing"
)

func TestGitStorageAndReader(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "forgehub-git-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("new storage: %v", err)
	}

	reader := NewReader()

	// 1. Path traversal should be rejected
	_, err = storage.GetDiskPath("users", "../hack", "repo")
	if err == nil {
		t.Errorf("expected path traversal error")
	}

	// 2. Initialize bare repository
	diskPath, err := storage.InitRepository("users", "alice", "project-x", "main")
	if err != nil {
		t.Fatalf("init repository: %v", err)
	}

	if !storage.RepoExists("users", "alice", "project-x") {
		t.Errorf("repo should exist on disk")
	}

	// 3. Create initial commit with README.md
	readmeText := "# Project X\n\nWelcome to Project X on ForgeHub!"
	err = storage.CreateInitialCommit(diskPath, "main", readmeText, "Alice", "alice@forgehub.local")
	if err != nil {
		t.Fatalf("create initial commit: %v", err)
	}

	// 4. Inspect branches
	branches, err := reader.ListBranches(diskPath)
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	if len(branches) == 0 || branches[0].Name != "main" {
		t.Errorf("expected main branch, got %+v", branches)
	}

	// 5. Inspect commits
	commits, err := reader.ListCommits(diskPath, "main", 10)
	if err != nil {
		t.Fatalf("list commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Message != "Initial commit" {
		t.Errorf("expected 'Initial commit', got '%s'", commits[0].Message)
	}

	// 6. Inspect tree
	tree, err := reader.ListTree(diskPath, "main", "")
	if err != nil {
		t.Fatalf("list tree: %v", err)
	}
	if len(tree) != 1 || tree[0].Name != "README.md" {
		t.Errorf("expected README.md in tree, got %+v", tree)
	}

	// 7. Inspect Readme
	readmeBlob, err := reader.GetReadme(diskPath, "main")
	if err != nil {
		t.Fatalf("get readme: %v", err)
	}
	if readmeBlob.Content != readmeText {
		t.Errorf("expected readme content '%s', got '%s'", readmeText, readmeBlob.Content)
	}
}

func TestSmartHTTPPacketLine(t *testing.T) {
	var buf stringsBuffer
	err := WritePacketLine(&buf, "# service=git-upload-pack\n")
	if err != nil {
		t.Fatalf("write packet line: %v", err)
	}
	expected := "001e# service=git-upload-pack\n"
	if buf.String() != expected {
		t.Errorf("expected '%s', got '%s'", expected, buf.String())
	}

	buf.Reset()
	err = WritePacketFlush(&buf)
	if err != nil {
		t.Fatalf("write flush: %v", err)
	}
	if buf.String() != "0000" {
		t.Errorf("expected '0000', got '%s'", buf.String())
	}
}

type stringsBuffer struct {
	data []byte
}

func (b *stringsBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *stringsBuffer) String() string {
	return string(b.data)
}

func (b *stringsBuffer) Reset() {
	b.data = b.data[:0]
}
