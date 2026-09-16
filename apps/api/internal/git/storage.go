package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrPathTraversal   = errors.New("invalid path: path traversal detected")
	ErrRepoNotFound    = errors.New("git repository not found on disk")
	ErrRepoAlreadyExists = errors.New("git repository already exists on disk")
)

type Storage interface {
	GetDiskPath(ownerType, ownerSlug, repoSlug string) (string, error)
	RepoExists(ownerType, ownerSlug, repoSlug string) bool
	InitRepository(ownerType, ownerSlug, repoSlug, defaultBranch string) (string, error)
	DeleteRepository(ownerType, ownerSlug, repoSlug string) error
	CreateInitialCommit(diskPath, defaultBranch, readmeContent, committerName, committerEmail string) error
	CheckMergeable(diskPath, baseBranch, headBranch string) (bool, error)
	MergeBranches(diskPath, baseBranch, headBranch, method, committerName, committerEmail, commitMsg string) (string, error)
}

type localStorage struct {
	rootDir string
}

func NewStorage(rootDir string) (Storage, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve git root: %w", err)
	}

	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return nil, fmt.Errorf("create git root: %w", err)
	}

	return &localStorage{rootDir: absRoot}, nil
}

func (s *localStorage) GetDiskPath(ownerType, ownerSlug, repoSlug string) (string, error) {
	// Clean and normalize components
	ownerType = strings.ToLower(strings.TrimSpace(ownerType))
	if ownerType != "users" && ownerType != "orgs" {
		ownerType = "users"
	}
	ownerSlug = strings.ToLower(strings.TrimSpace(ownerSlug))
	repoSlug = strings.ToLower(strings.TrimSpace(repoSlug))

	// Security: Block any directory traversal sequences
	if strings.Contains(ownerSlug, "..") || strings.Contains(ownerSlug, "/") || strings.Contains(ownerSlug, "\\") ||
		strings.Contains(repoSlug, "..") || strings.Contains(repoSlug, "/") || strings.Contains(repoSlug, "\\") {
		return "", ErrPathTraversal
	}

	if !strings.HasSuffix(repoSlug, ".git") {
		repoSlug = repoSlug + ".git"
	}

	// Construct target path
	targetPath := filepath.Join(s.rootDir, ownerType, ownerSlug, repoSlug)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	// Security: Prevent path traversal
	cleanRoot := filepath.Clean(s.rootDir) + string(filepath.Separator)
	cleanAbs := filepath.Clean(absPath)
	if !strings.HasPrefix(cleanAbs+string(filepath.Separator), cleanRoot) && cleanAbs != filepath.Clean(s.rootDir) {
		return "", ErrPathTraversal
	}

	return absPath, nil
}

func (s *localStorage) RepoExists(ownerType, ownerSlug, repoSlug string) bool {
	path, err := s.GetDiskPath(ownerType, ownerSlug, repoSlug)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	// Verify it contains HEAD
	headFile := filepath.Join(path, "HEAD")
	if _, err := os.Stat(headFile); err == nil {
		return true
	}
	return false
}

func (s *localStorage) InitRepository(ownerType, ownerSlug, repoSlug, defaultBranch string) (string, error) {
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	diskPath, err := s.GetDiskPath(ownerType, ownerSlug, repoSlug)
	if err != nil {
		return "", err
	}

	if s.RepoExists(ownerType, ownerSlug, repoSlug) {
		return "", ErrRepoAlreadyExists
	}

	if err := os.MkdirAll(diskPath, 0755); err != nil {
		return "", fmt.Errorf("create repo dir: %w", err)
	}

	// Run git init --bare -b <defaultBranch>
	cmd := exec.Command("git", "init", "--bare", "-b", defaultBranch, diskPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Fallback for older git versions without -b flag
		cmdFallback := exec.Command("git", "init", "--bare", diskPath)
		cmdFallback.Stderr = &stderr
		if err2 := cmdFallback.Run(); err2 != nil {
			return "", fmt.Errorf("git init failed: %s: %w", stderr.String(), err2)
		}
		// Set default branch in HEAD
		headPath := filepath.Join(diskPath, "HEAD")
		_ = os.WriteFile(headPath, []byte(fmt.Sprintf("ref: refs/heads/%s\n", defaultBranch)), 0644)
	}

	// Enable HTTP receivepack (allow push)
	cfgCmd := exec.Command("git", "-C", diskPath, "config", "http.receivepack", "true")
	_ = cfgCmd.Run()

	return diskPath, nil
}

func (s *localStorage) DeleteRepository(ownerType, ownerSlug, repoSlug string) error {
	diskPath, err := s.GetDiskPath(ownerType, ownerSlug, repoSlug)
	if err != nil {
		return err
	}
	return os.RemoveAll(diskPath)
}

func (s *localStorage) CreateInitialCommit(diskPath, defaultBranch, readmeContent, committerName, committerEmail string) error {
	if committerName == "" {
		committerName = "ForgeHub"
	}
	if committerEmail == "" {
		committerEmail = "noreply@forgehub.local"
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// 1. Write blob: git hash-object -w --stdin
	hashCmd := exec.Command("git", "-C", diskPath, "hash-object", "-w", "--stdin")
	hashCmd.Stdin = strings.NewReader(readmeContent)
	blobOut, err := hashCmd.Output()
	if err != nil {
		return fmt.Errorf("create blob: %w", err)
	}
	blobSHA := strings.TrimSpace(string(blobOut))

	// 2. Create tree: git mktree
	treeEntry := fmt.Sprintf("100644 blob %s\tREADME.md\n", blobSHA)
	mktreeCmd := exec.Command("git", "-C", diskPath, "mktree")
	mktreeCmd.Stdin = strings.NewReader(treeEntry)
	treeOut, err := mktreeCmd.Output()
	if err != nil {
		return fmt.Errorf("create tree: %w", err)
	}
	treeSHA := strings.TrimSpace(string(treeOut))

	// 3. Create commit: git commit-tree <treeSHA>
	commitCmd := exec.Command("git", "-C", diskPath, "commit-tree", treeSHA)
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+committerName,
		"GIT_AUTHOR_EMAIL="+committerEmail,
		"GIT_AUTHOR_DATE="+time.Now().UTC().Format(time.RFC3339),
		"GIT_COMMITTER_NAME="+committerName,
		"GIT_COMMITTER_EMAIL="+committerEmail,
		"GIT_COMMITTER_DATE="+time.Now().UTC().Format(time.RFC3339),
	)
	commitCmd.Stdin = strings.NewReader("Initial commit\n")
	commitOut, err := commitCmd.Output()
	if err != nil {
		return fmt.Errorf("create commit: %w", err)
	}
	commitSHA := strings.TrimSpace(string(commitOut))

	// 4. Update ref: git update-ref refs/heads/<defaultBranch> <commitSHA>
	refPath := "refs/heads/" + defaultBranch
	updateRefCmd := exec.Command("git", "-C", diskPath, "update-ref", refPath, commitSHA)
	if err := updateRefCmd.Run(); err != nil {
		return fmt.Errorf("update ref: %w", err)
	}

	// 5. Ensure HEAD points to the branch
	headPath := filepath.Join(diskPath, "HEAD")
	_ = os.WriteFile(headPath, []byte(fmt.Sprintf("ref: %s\n", refPath)), 0644)

	return nil
}

func (s *localStorage) CheckMergeable(diskPath, baseBranch, headBranch string) (bool, error) {
	cmd := exec.Command("git", "-C", diskPath, "merge-tree", "--write-tree", baseBranch, headBranch)
	err := cmd.Run()
	if err != nil {
		return false, nil // Has conflicts
	}
	return true, nil
}

func (s *localStorage) MergeBranches(diskPath, baseBranch, headBranch, method, committerName, committerEmail, commitMsg string) (string, error) {
	if committerName == "" {
		committerName = "ForgeHub"
	}
	if committerEmail == "" {
		committerEmail = "noreply@forgehub.local"
	}
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Merge branch '%s' into %s", headBranch, baseBranch)
	}

	// 1. Get merged tree SHA
	treeCmd := exec.Command("git", "-C", diskPath, "merge-tree", "--write-tree", baseBranch, headBranch)
	treeOut, err := treeCmd.Output()
	if err != nil {
		return "", errors.New("cannot merge: branches have conflicts")
	}
	treeSHA := strings.TrimSpace(string(treeOut))

	// 2. Get base and head commit SHAs
	baseCommitCmd := exec.Command("git", "-C", diskPath, "rev-parse", baseBranch)
	baseOut, err := baseCommitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve base commit: %w", err)
	}
	baseSHA := strings.TrimSpace(string(baseOut))

	headCommitCmd := exec.Command("git", "-C", diskPath, "rev-parse", headBranch)
	headOut, err := headCommitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve head commit: %w", err)
	}
	headSHA := strings.TrimSpace(string(headOut))

	// 3. Create merge commit
	var commitArgs []string
	if method == "squash" {
		// Squash merge: single parent (baseSHA)
		commitArgs = []string{"-C", diskPath, "commit-tree", treeSHA, "-p", baseSHA}
	} else {
		// Standard merge commit: two parents (baseSHA and headSHA)
		commitArgs = []string{"-C", diskPath, "commit-tree", treeSHA, "-p", baseSHA, "-p", headSHA}
	}

	commitCmd := exec.Command("git", commitArgs...)
	commitCmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+committerName,
		"GIT_AUTHOR_EMAIL="+committerEmail,
		"GIT_AUTHOR_DATE="+time.Now().UTC().Format(time.RFC3339),
		"GIT_COMMITTER_NAME="+committerName,
		"GIT_COMMITTER_EMAIL="+committerEmail,
		"GIT_COMMITTER_DATE="+time.Now().UTC().Format(time.RFC3339),
	)
	commitCmd.Stdin = strings.NewReader(commitMsg + "\n")
	commitOut, err := commitCmd.Output()
	if err != nil {
		return "", fmt.Errorf("commit-tree failed: %w", err)
	}
	mergeCommitSHA := strings.TrimSpace(string(commitOut))

	// 4. Update base branch ref
	refPath := "refs/heads/" + baseBranch
	updateRefCmd := exec.Command("git", "-C", diskPath, "update-ref", refPath, mergeCommitSHA)
	if err := updateRefCmd.Run(); err != nil {
		return "", fmt.Errorf("update-ref failed: %w", err)
	}

	return mergeCommitSHA, nil
}
