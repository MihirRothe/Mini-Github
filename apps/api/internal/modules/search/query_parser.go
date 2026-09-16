package search

import (
	"strings"
)

// ParseQuery takes a raw search string (e.g. "auth failure repo:octocat/hello is:open author:alice")
// and decomposes it into structured filters and clean free-text tokens.
func ParseQuery(raw string) ParsedQuery {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ParsedQuery{Raw: ""}
	}

	tokens := strings.Fields(trimmed)
	var cleanTokens []string

	pq := ParsedQuery{
		Raw: trimmed,
	}

	for _, token := range tokens {
		lower := strings.ToLower(token)

		switch {
		case strings.HasPrefix(lower, "repo:"):
			repoSpec := token[5:]
			if strings.Contains(repoSpec, "/") {
				parts := strings.SplitN(repoSpec, "/", 2)
				pq.RepoOwner = parts[0]
				pq.RepoSlug = parts[1]
			} else {
				pq.RepoSlug = repoSpec
			}

		case strings.HasPrefix(lower, "author:"):
			pq.Author = token[7:]

		case strings.HasPrefix(lower, "state:"):
			val := lower[6:]
			if val == "open" || val == "closed" || val == "merged" {
				pq.State = val
			}

		case strings.HasPrefix(lower, "is:"):
			val := lower[3:]
			switch val {
			case "open", "closed", "merged":
				pq.State = val
			case "pr":
				isPR := true
				pq.IsPR = &isPR
			case "issue":
				isIssue := true
				pq.IsIssue = &isIssue
			}

		case strings.HasPrefix(lower, "lang:") || strings.HasPrefix(lower, "language:"):
			idx := strings.Index(token, ":")
			pq.Language = token[idx+1:]

		case strings.HasPrefix(lower, "sort:"):
			pq.Sort = lower[5:]

		default:
			cleanTokens = append(cleanTokens, token)
		}
	}

	pq.CleanQuery = strings.Join(cleanTokens, " ")
	return pq
}
