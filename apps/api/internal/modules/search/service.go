package search

import (
	"context"
	"path/filepath"
	"strings"

	"forgehub/apps/api/internal/git"
)

type Service interface {
	Search(ctx context.Context, currentUserID string, isSiteAdmin bool, rawQuery string, searchType SearchType, page, perPage int) (*SearchResults, error)
	QuickSearch(ctx context.Context, currentUserID string, isSiteAdmin bool, rawQuery string) (*QuickSearchResponse, error)
}

type service struct {
	store     SearchStore
	gitReader git.Reader
}

func NewService(store SearchStore, gitReader git.Reader) Service {
	return &service{
		store:     store,
		gitReader: gitReader,
	}
}

func (s *service) Search(ctx context.Context, currentUserID string, isSiteAdmin bool, rawQuery string, searchType SearchType, page, perPage int) (*SearchResults, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	pq := ParseQuery(rawQuery)

	// If query explicitly sets `is:pr` or `is:issue`, override searchType if default
	if searchType == "" || searchType == SearchTypeAll {
		if pq.IsPR != nil && *pq.IsPR {
			searchType = SearchTypePulls
		} else if pq.IsIssue != nil && *pq.IsIssue {
			searchType = SearchTypeIssues
		} else {
			searchType = SearchTypeRepositories
		}
	}

	results := &SearchResults{
		Query:   rawQuery,
		Type:    searchType,
		Page:    page,
		PerPage: perPage,
	}

	// Calculate counts across tabs for the sidebar
	_, repoTotal, _ := s.store.SearchRepositories(ctx, currentUserID, isSiteAdmin, pq, 1, 1)
	_, issueTotal, _ := s.store.SearchIssues(ctx, currentUserID, isSiteAdmin, pq, 1, 1)
	_, pullTotal, _ := s.store.SearchPulls(ctx, currentUserID, isSiteAdmin, pq, 1, 1)
	_, userTotal, _ := s.store.SearchUsers(ctx, pq, 1, 1)

	results.Counts.Repositories = repoTotal
	results.Counts.Issues = issueTotal
	results.Counts.Pulls = pullTotal
	results.Counts.Users = userTotal

	switch searchType {
	case SearchTypeRepositories:
		items, total, err := s.store.SearchRepositories(ctx, currentUserID, isSiteAdmin, pq, page, perPage)
		if err != nil {
			return nil, err
		}
		results.Repositories = items
		results.TotalCount = total

	case SearchTypeIssues:
		items, total, err := s.store.SearchIssues(ctx, currentUserID, isSiteAdmin, pq, page, perPage)
		if err != nil {
			return nil, err
		}
		results.Issues = items
		results.TotalCount = total

	case SearchTypePulls:
		items, total, err := s.store.SearchPulls(ctx, currentUserID, isSiteAdmin, pq, page, perPage)
		if err != nil {
			return nil, err
		}
		results.Pulls = items
		results.TotalCount = total

	case SearchTypeCode:
		codeItems, total, err := s.searchCode(ctx, currentUserID, isSiteAdmin, pq, page, perPage)
		if err != nil {
			return nil, err
		}
		results.Code = codeItems
		results.TotalCount = total
		results.Counts.Code = total

	case SearchTypeUsers:
		items, total, err := s.store.SearchUsers(ctx, pq, page, perPage)
		if err != nil {
			return nil, err
		}
		results.Users = items
		results.TotalCount = total
	}

	return results, nil
}

func (s *service) QuickSearch(ctx context.Context, currentUserID string, isSiteAdmin bool, rawQuery string) (*QuickSearchResponse, error) {
	pq := ParseQuery(rawQuery)

	resp := &QuickSearchResponse{
		Query:        rawQuery,
		Repositories: []RepoResultItem{},
		Issues:       []IssueResultItem{},
		Pulls:        []PRResultItem{},
		Code:         []CodeResultItem{},
		Users:        []UserResultItem{},
	}

	if strings.TrimSpace(rawQuery) == "" {
		return resp, nil
	}

	// Quick repos (top 4)
	if repos, _, err := s.store.SearchRepositories(ctx, currentUserID, isSiteAdmin, pq, 1, 4); err == nil {
		resp.Repositories = repos
	}

	// Quick issues (top 3)
	if issues, _, err := s.store.SearchIssues(ctx, currentUserID, isSiteAdmin, pq, 1, 3); err == nil {
		resp.Issues = issues
	}

	// Quick PRs (top 3)
	if pulls, _, err := s.store.SearchPulls(ctx, currentUserID, isSiteAdmin, pq, 1, 3); err == nil {
		resp.Pulls = pulls
	}

	// Quick users (top 3)
	if users, _, err := s.store.SearchUsers(ctx, pq, 1, 3); err == nil {
		resp.Users = users
	}

	// Quick code (top 3 files)
	if code, _, err := s.searchCode(ctx, currentUserID, isSiteAdmin, pq, 1, 3); err == nil {
		resp.Code = code
	}

	return resp, nil
}

func (s *service) searchCode(ctx context.Context, currentUserID string, isSiteAdmin bool, pq ParsedQuery, page, perPage int) ([]CodeResultItem, int, error) {
	if pq.CleanQuery == "" {
		return []CodeResultItem{}, 0, nil
	}

	repos, err := s.store.ListAccessibleReposForCodeSearch(ctx, currentUserID, isSiteAdmin, pq.RepoOwner, pq.RepoSlug)
	if err != nil {
		return nil, 0, err
	}

	var allItems []CodeResultItem
	for _, r := range repos {
		if r.DiskPath == "" || s.gitReader == nil {
			continue
		}

		matches, err := s.gitReader.Grep(r.DiskPath, "HEAD", pq.CleanQuery, 20)
		if err != nil || len(matches) == 0 {
			continue
		}

		// Group matches by file
		fileMatches := make(map[string][]git.GrepMatch)
		var fileOrder []string
		for _, m := range matches {
			if pq.Language != "" {
				ext := strings.TrimPrefix(filepath.Ext(m.Path), ".")
				if !strings.EqualFold(ext, pq.Language) {
					continue
				}
			}
			if _, exists := fileMatches[m.Path]; !exists {
				fileOrder = append(fileOrder, m.Path)
			}
			fileMatches[m.Path] = append(fileMatches[m.Path], m)
		}

		for _, filePath := range fileOrder {
			allItems = append(allItems, CodeResultItem{
				RepoOwner: r.OwnerName,
				RepoSlug:  r.RepoSlug,
				FilePath:  filePath,
				Matches:   fileMatches[filePath],
			})
		}
	}

	total := len(allItems)
	start := (page - 1) * perPage
	if start >= total {
		return []CodeResultItem{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}

	return allItems[start:end], total, nil
}
