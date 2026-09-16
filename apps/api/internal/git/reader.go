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

type DiffLine struct {
	Type    string `json:"type"` // "context", "add", "del"
	Content string `json:"content"`
	OldNum  int    `json:"old_num,omitempty"`
	NewNum  int    `json:"new_num,omitempty"`
}

type DiffHunk struct {
	Header string     `json:"header"`
	Lines  []DiffLine `json:"lines"`
}

type DiffFile struct {
	OldPath   string     `json:"old_path"`
	NewPath   string     `json:"new_path"`
	Status    string     `json:"status"` // "added", "modified", "deleted"
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	Hunks     []DiffHunk `json:"hunks"`
	Patch     string     `json:"patch"`
}

type DiffResult struct {
	BaseRef        string      `json:"base_ref"`
	HeadRef        string      `json:"head_ref"`
	Files          []*DiffFile `json:"files"`
	TotalAdditions int         `json:"total_additions"`
	TotalDeletions int         `json:"total_deletions"`
	FilesChanged   int         `json:"files_changed"`
}

type Reader interface {
	GetDefaultBranch(diskPath string) (string, error)
	ListBranches(diskPath string) ([]BranchInfo, error)
	ListCommits(diskPath, ref string, limit int) ([]CommitInfo, error)
	GetCommit(diskPath, hash string) (*CommitInfo, error)
	ListTree(diskPath, ref, subPath string) ([]TreeEntry, error)
	GetBlob(diskPath, ref, filePath string) (*BlobInfo, error)
	GetReadme(diskPath, ref string) (*BlobInfo, error)
	DiffBranches(diskPath, baseRef, headRef string) (*DiffResult, error)
	GetCommitsBetween(diskPath, baseRef, headRef string) ([]CommitInfo, error)
	Grep(diskPath, ref, query string, maxResults int) ([]GrepMatch, error)
}

type GrepMatch struct {
	Path       string `json:"path"`
	LineNumber int    `json:"line_number"`
	LineText   string `json:"line_text"`
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

func (r *localReader) DiffBranches(diskPath, baseRef, headRef string) (*DiffResult, error) {
	cmd := exec.Command("git", "-C", diskPath, "diff", "-p", "-U3", fmt.Sprintf("%s...%s", baseRef, headRef))
	out, err := cmd.Output()
	if err != nil {
		return &DiffResult{BaseRef: baseRef, HeadRef: headRef, Files: []*DiffFile{}}, nil
	}

	result := &DiffResult{
		BaseRef: baseRef,
		HeadRef: headRef,
		Files:   []*DiffFile{},
	}

	rawPatch := string(out)
	if strings.TrimSpace(rawPatch) == "" {
		return result, nil
	}

	fileBlocks := strings.Split(rawPatch, "diff --git ")
	for _, block := range fileBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		headerParts := strings.Fields(lines[0])
		if len(headerParts) < 2 {
			continue
		}

		oldPath := strings.TrimPrefix(headerParts[0], "a/")
		newPath := strings.TrimPrefix(headerParts[1], "b/")
		status := "modified"

		diffFile := &DiffFile{
			OldPath: oldPath,
			NewPath: newPath,
			Status:  status,
			Hunks:   []DiffHunk{},
			Patch:   "diff --git " + block,
		}

		var currentHunk *DiffHunk
		oldLineNum := 0
		newLineNum := 0

		for _, line := range lines[1:] {
			if strings.HasPrefix(line, "new file mode") {
				diffFile.Status = "added"
			} else if strings.HasPrefix(line, "deleted file mode") {
				diffFile.Status = "deleted"
			} else if strings.HasPrefix(line, "@@") {
				if currentHunk != nil {
					diffFile.Hunks = append(diffFile.Hunks, *currentHunk)
				}
				currentHunk = &DiffHunk{
					Header: line,
					Lines:  []DiffLine{},
				}
				hunkParts := strings.Split(line, " ")
				if len(hunkParts) >= 3 {
					oldPart := strings.TrimPrefix(hunkParts[1], "-")
					newPart := strings.TrimPrefix(hunkParts[2], "+")
					oldNums := strings.Split(oldPart, ",")
					newNums := strings.Split(newPart, ",")
					oldLineNum, _ = strconv.Atoi(oldNums[0])
					newLineNum, _ = strconv.Atoi(newNums[0])
				}
			} else if currentHunk != nil {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					diffFile.Additions++
					result.TotalAdditions++
					currentHunk.Lines = append(currentHunk.Lines, DiffLine{
						Type:    "add",
						Content: line[1:],
						NewNum:  newLineNum,
					})
					newLineNum++
				} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					diffFile.Deletions++
					result.TotalDeletions++
					currentHunk.Lines = append(currentHunk.Lines, DiffLine{
						Type:    "del",
						Content: line[1:],
						OldNum:  oldLineNum,
					})
					oldLineNum++
				} else if strings.HasPrefix(line, " ") {
					currentHunk.Lines = append(currentHunk.Lines, DiffLine{
						Type:    "context",
						Content: line[1:],
						OldNum:  oldLineNum,
						NewNum:  newLineNum,
					})
					oldLineNum++
					newLineNum++
				}
			}
		}

		if currentHunk != nil {
			diffFile.Hunks = append(diffFile.Hunks, *currentHunk)
		}

		result.Files = append(result.Files, diffFile)
	}

	result.FilesChanged = len(result.Files)
	return result, nil
}

func (r *localReader) GetCommitsBetween(diskPath, baseRef, headRef string) ([]CommitInfo, error) {
	cmd := exec.Command("git", "-C", diskPath, "log", "--format=%H|%h|%an|%ae|%at|%s", fmt.Sprintf("%s..%s", baseRef, headRef))
	out, err := cmd.Output()
	if err != nil {
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

func (r *localReader) Grep(diskPath, ref, query string, maxResults int) ([]GrepMatch, error) {
	if query == "" {
		return []GrepMatch{}, nil
	}
	if ref == "" {
		ref = "HEAD"
	}
	if maxResults <= 0 || maxResults > 200 {
		maxResults = 50
	}

	// git -C <diskPath> grep -n -I -i -F -m <maxResults> -e <query> <ref>
	args := []string{
		"-C", diskPath,
		"grep",
		"-n",
		"-I",
		"-i",
		"-F",
		"-m", strconv.Itoa(maxResults),
		"-e", query,
		ref,
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		// Exit status 1 means no matches found; other errors might be empty repo or missing ref
		return []GrepMatch{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var matches []GrepMatch
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: <ref>:<filePath>:<lineNum>:<content>
		parts := strings.SplitN(line, ":", 4)
		if len(parts) >= 4 {
			lineNum, err := strconv.Atoi(parts[2])
			if err != nil {
				continue
			}
			matches = append(matches, GrepMatch{
				Path:       parts[1],
				LineNumber: lineNum,
				LineText:   parts[3],
			})
		}
		if len(matches) >= maxResults {
			break
		}
	}

	return matches, nil
}
