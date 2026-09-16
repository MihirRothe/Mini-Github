package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrObjectNotFound = errors.New("git object not found")
	ErrRefNotFound    = errors.New("git reference not found")
)

type BranchInfo struct {
	Name       string `json:"name"`
	CommitHash string `json:"commit_hash"`
	IsDefault  bool   `json:"is_default"`
}

type CommitInfo struct {
	Hash        string    `json:"hash"`
	ShortHash   string    `json:"short_hash"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	AuthorDate  time.Time `json:"author_date"`
	Message     string    `json:"message"`
}

type TreeEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	Mode string `json:"mode"`
	Size int64  `json:"size"`
}

type BlobInfo struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	IsBinary bool   `json:"is_binary"`
}

type Reader interface {
	GetDefaultBranch(diskPath string) (string, error)
	ListBranches(diskPath string) ([]BranchInfo, error)
	ListCommits(diskPath, ref string, limit int) ([]CommitInfo, error)
	GetCommit(diskPath, hash string) (*CommitInfo, error)
	ListTree(diskPath, ref, subPath string) ([]TreeEntry, error)
	GetBlob(diskPath, ref, filePath string) (*BlobInfo, error)
	GetReadme(diskPath, ref string) (*BlobInfo, error)
}

type localReader struct{}

func NewReader() Reader {
	return &localReader{}
}

func (r *localReader) GetDefaultBranch(diskPath string) (string, error) {
	cmd := exec.Command("git", "-C", diskPath, "symbolic-ref", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "main", nil // Default fallback
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "main", nil
	}
	return branch, nil
}

func (r *localReader) ListBranches(diskPath string) ([]BranchInfo, error) {
	defaultBranch, _ := r.GetDefaultBranch(diskPath)

	cmd := exec.Command("git", "-C", diskPath, "for-each-ref", "--format=%(refname:short)|%(objectname)|%(HEAD)", "refs/heads/")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var branches []BranchInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			name := parts[0]
			hash := parts[1]
			isDef := name == defaultBranch || (len(parts) >= 3 && parts[2] == "*")
			branches = append(branches, BranchInfo{
				Name:       name,
				CommitHash: hash,
				IsDefault:  isDef,
			})
		}
	}

	// If no branches found (empty bare repo), synthesize default branch
	if len(branches) == 0 {
		branches = append(branches, BranchInfo{
			Name:       defaultBranch,
			CommitHash: "",
			IsDefault:  true,
		})
	}

	return branches, nil
}

func (r *localReader) ListCommits(diskPath, ref string, limit int) ([]CommitInfo, error) {
	if ref == "" {
		ref = "HEAD"
	}
	if limit <= 0 {
		limit = 30
	}

	cmd := exec.Command("git", "-C", diskPath, "log", fmt.Sprintf("-n%d", limit), "--format=%H|%h|%an|%ae|%at|%s", ref)
	out, err := cmd.Output()
	if err != nil {
		// Empty repo or invalid ref
		return []CommitInfo{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var commits []CommitInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 6)
		if len(parts) >= 6 {
			sec, _ := strconv.ParseInt(parts[4], 10, 64)
			commits = append(commits, CommitInfo{
				Hash:        parts[0],
				ShortHash:   parts[1],
				AuthorName:  parts[2],
				AuthorEmail: parts[3],
				AuthorDate:  time.Unix(sec, 0).UTC(),
				Message:     parts[5],
			})
		}
	}
	return commits, nil
}

func (r *localReader) GetCommit(diskPath, hash string) (*CommitInfo, error) {
	cmd := exec.Command("git", "-C", diskPath, "show", "-s", "--format=%H|%h|%an|%ae|%at|%s", hash)
	out, err := cmd.Output()
	if err != nil {
		return nil, ErrObjectNotFound
	}
	line := strings.TrimSpace(string(out))
	parts := strings.SplitN(line, "|", 6)
	if len(parts) < 6 {
		return nil, ErrObjectNotFound
	}
	sec, _ := strconv.ParseInt(parts[4], 10, 64)
	return &CommitInfo{
		Hash:        parts[0],
		ShortHash:   parts[1],
		AuthorName:  parts[2],
		AuthorEmail: parts[3],
		AuthorDate:  time.Unix(sec, 0).UTC(),
		Message:     parts[5],
	}, nil
}

func (r *localReader) ListTree(diskPath, ref, subPath string) ([]TreeEntry, error) {
	if ref == "" {
		ref = "HEAD"
	}

	treeTarget := ref
	subPath = strings.Trim(subPath, "/")
	if subPath != "" {
		treeTarget = fmt.Sprintf("%s:%s", ref, subPath)
	}

	cmd := exec.Command("git", "-C", diskPath, "ls-tree", "-l", treeTarget)
	out, err := cmd.Output()
	if err != nil {
		return []TreeEntry{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var entries []TreeEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: <mode> <type> <object> <size>\t<file>
		tabParts := strings.SplitN(line, "\t", 2)
		if len(tabParts) != 2 {
			continue
		}
		filename := tabParts[1]
		metaParts := strings.Fields(tabParts[0])
		if len(metaParts) < 4 {
			continue
		}

		mode := metaParts[0]
		entryType := metaParts[1]
		var size int64
		if metaParts[3] != "-" {
			size, _ = strconv.ParseInt(metaParts[3], 10, 64)
		}

		entryPath := filename
		if subPath != "" {
			entryPath = subPath + "/" + filename
		}

		entries = append(entries, TreeEntry{
			Name: filename,
			Path: entryPath,
			Type: entryType,
			Mode: mode,
			Size: size,
		})
	}

	// Sort trees first, then alphabetically
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == "tree"
		}
		return entries[i].Name < entries[j].Name
	})

	return entries, nil
}

func (r *localReader) GetBlob(diskPath, ref, filePath string) (*BlobInfo, error) {
	if ref == "" {
		ref = "HEAD"
	}
	filePath = strings.Trim(filePath, "/")
	target := fmt.Sprintf("%s:%s", ref, filePath)

	cmd := exec.Command("git", "-C", diskPath, "cat-file", "-p", target)
	out, err := cmd.Output()
	if err != nil {
		return nil, ErrObjectNotFound
	}

	isBinary := bytes.IndexByte(out[:min(len(out), 8000)], 0) != -1
	content := ""
	if !isBinary {
		content = string(out)
	}

	return &BlobInfo{
		Path:     filePath,
		Name:     filepath.Base(filePath),
		Content:  content,
		Size:     int64(len(out)),
		IsBinary: isBinary,
	}, nil
}

func (r *localReader) GetReadme(diskPath, ref string) (*BlobInfo, error) {
	candidates := []string{"README.md", "readme.md", "README", "readme.txt", "README.txt"}
	for _, name := range candidates {
		blob, err := r.GetBlob(diskPath, ref, name)
		if err == nil && blob != nil {
			return blob, nil
		}
	}
	return nil, ErrObjectNotFound
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
