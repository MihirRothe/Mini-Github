import React, { useState, useEffect } from 'react';
import { useSearchParams, Link, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  Search,
  FolderGit2,
  AlertCircle,
  GitPullRequest,
  Code as CodeIcon,
  User as UserIcon,
  Building2,
  Star,
  GitFork,
  MessageSquare,
  Clock,
  ArrowRight,
  Filter,
  CheckCircle2
} from 'lucide-react';
import {
  SearchType,
  SearchResultsResponse
} from '../types';

export const SearchPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();

  const queryParam = searchParams.get('q') || '';
  const typeParam = (searchParams.get('type') as SearchType) || 'repositories';
  const pageParam = parseInt(searchParams.get('page') || '1', 10);
  const sortParam = searchParams.get('sort') || '';

  const [inputQuery, setInputQuery] = useState(queryParam);

  useEffect(() => {
    setInputQuery(queryParam);
  }, [queryParam]);

  const { data, isLoading, isError } = useQuery<SearchResultsResponse>({
    queryKey: ['search', queryParam, typeParam, pageParam, sortParam],
    queryFn: async () => {
      if (!queryParam.trim()) {
        return {
          query: '',
          type: typeParam,
          total_count: 0,
          page: 1,
          per_page: 20,
          counts: { repositories: 0, issues: 0, pulls: 0, code: 0, users: 0 },
        };
      }
      const params = new URLSearchParams();
      params.set('q', queryParam);
      params.set('type', typeParam);
      params.set('page', pageParam.toString());
      if (sortParam) params.set('sort', sortParam);

      const res = await fetch(`/api/v1/search?${params.toString()}`);
      if (!res.ok) throw new Error('Search failed');
      return res.json();
    },
    enabled: true,
  });

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const next = new URLSearchParams(searchParams);
    next.set('q', inputQuery.trim());
    next.set('page', '1');
    setSearchParams(next);
  };

  const handleTypeChange = (newType: SearchType) => {
    const next = new URLSearchParams(searchParams);
    next.set('type', newType);
    next.set('page', '1');
    setSearchParams(next);
  };

  const handleSortChange = (newSort: string) => {
    const next = new URLSearchParams(searchParams);
    if (newSort) {
      next.set('sort', newSort);
    } else {
      next.delete('sort');
    }
    next.set('page', '1');
    setSearchParams(next);
  };

  const handlePageChange = (newPage: number) => {
    const next = new URLSearchParams(searchParams);
    next.set('page', newPage.toString());
    setSearchParams(next);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  };

  const appendQualifier = (qualifier: string) => {
    if (!inputQuery.includes(qualifier)) {
      const updated = inputQuery.trim() ? `${inputQuery.trim()} ${qualifier}` : qualifier;
      setInputQuery(updated);
      const next = new URLSearchParams(searchParams);
      next.set('q', updated);
      next.set('page', '1');
      setSearchParams(next);
    }
  };

  const counts = data?.counts || { repositories: 0, issues: 0, pulls: 0, code: 0, users: 0 };
  const totalCount = data?.total_count || 0;
  const totalPages = Math.ceil(totalCount / 20) || 1;

  return (
    <div className="space-y-6 animate-fade-in pb-12">
      {/* Search Header Form */}
      <div className="bg-forge-card border border-forge-border rounded-xl p-6 shadow-sm">
        <form onSubmit={handleSearchSubmit} className="space-y-4">
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Search className="w-5 h-5 text-forge-muted absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
              <input
                type="text"
                value={inputQuery}
                onChange={e => setInputQuery(e.target.value)}
                placeholder="Search ForgeHub with text, repo:owner/name, author:user, is:open, lang:go..."
                className="w-full pl-11 pr-4 py-2.5 rounded-lg bg-forge-bg border border-forge-border text-forge-text text-sm focus:outline-none focus:border-forge-accent transition-colors shadow-inner"
              />
            </div>
            <button
              type="submit"
              className="px-6 py-2.5 bg-forge-accent text-white font-medium text-sm rounded-lg hover:bg-forge-accent/90 transition-colors shadow-sm flex items-center gap-2"
            >
              <span>Search</span>
            </button>
          </div>

          {/* Quick Qualifier Helper Chips */}
          <div className="flex flex-wrap items-center gap-2 text-xs text-forge-muted pt-1">
            <span className="flex items-center gap-1 text-forge-subtle">
              <Filter className="w-3.5 h-3.5" />
              Filters:
            </span>
            <button
              type="button"
              onClick={() => appendQualifier('is:open')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              is:open
            </button>
            <button
              type="button"
              onClick={() => appendQualifier('is:closed')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              is:closed
            </button>
            <button
              type="button"
              onClick={() => appendQualifier('is:pr')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              is:pr
            </button>
            <button
              type="button"
              onClick={() => appendQualifier('is:issue')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              is:issue
            </button>
            <button
              type="button"
              onClick={() => appendQualifier('lang:go')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              lang:go
            </button>
            <button
              type="button"
              onClick={() => appendQualifier('lang:ts')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              lang:ts
            </button>
            <button
              type="button"
              onClick={() => appendQualifier('sort:stars')}
              className="px-2 py-0.5 rounded bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text transition-colors"
            >
              sort:stars
            </button>
          </div>
        </form>
      </div>

      {/* Main Grid: Sidebar Types + Results */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 items-start">
        {/* Left Sidebar: Type Tabs */}
        <div className="bg-forge-card border border-forge-border rounded-xl p-2 space-y-1 shadow-sm md:sticky md:top-20">
          <button
            onClick={() => handleTypeChange('repositories')}
            className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
              typeParam === 'repositories'
                ? 'bg-forge-accent text-white'
                : 'text-forge-text hover:bg-forge-surface'
            }`}
          >
            <div className="flex items-center space-x-2.5">
              <FolderGit2 className="w-4 h-4" />
              <span>Repositories</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-mono ${
                typeParam === 'repositories'
                  ? 'bg-white/20 text-white'
                  : 'bg-forge-surface text-forge-muted border border-forge-border'
              }`}
            >
              {counts.repositories}
            </span>
          </button>

          <button
            onClick={() => handleTypeChange('code')}
            className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
              typeParam === 'code'
                ? 'bg-forge-accent text-white'
                : 'text-forge-text hover:bg-forge-surface'
            }`}
          >
            <div className="flex items-center space-x-2.5">
              <CodeIcon className="w-4 h-4" />
              <span>Code</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-mono ${
                typeParam === 'code'
                  ? 'bg-white/20 text-white'
                  : 'bg-forge-surface text-forge-muted border border-forge-border'
              }`}
            >
              {counts.code}
            </span>
          </button>

          <button
            onClick={() => handleTypeChange('issues')}
            className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
              typeParam === 'issues'
                ? 'bg-forge-accent text-white'
                : 'text-forge-text hover:bg-forge-surface'
            }`}
          >
            <div className="flex items-center space-x-2.5">
              <AlertCircle className="w-4 h-4" />
              <span>Issues</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-mono ${
                typeParam === 'issues'
                  ? 'bg-white/20 text-white'
                  : 'bg-forge-surface text-forge-muted border border-forge-border'
              }`}
            >
              {counts.issues}
            </span>
          </button>

          <button
            onClick={() => handleTypeChange('pulls')}
            className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
              typeParam === 'pulls'
                ? 'bg-forge-accent text-white'
                : 'text-forge-text hover:bg-forge-surface'
            }`}
          >
            <div className="flex items-center space-x-2.5">
              <GitPullRequest className="w-4 h-4" />
              <span>Pull Requests</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-mono ${
                typeParam === 'pulls'
                  ? 'bg-white/20 text-white'
                  : 'bg-forge-surface text-forge-muted border border-forge-border'
              }`}
            >
              {counts.pulls}
            </span>
          </button>

          <button
            onClick={() => handleTypeChange('users')}
            className={`w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
              typeParam === 'users'
                ? 'bg-forge-accent text-white'
                : 'text-forge-text hover:bg-forge-surface'
            }`}
          >
            <div className="flex items-center space-x-2.5">
              <UserIcon className="w-4 h-4" />
              <span>Users &amp; Orgs</span>
            </div>
            <span
              className={`text-xs px-2 py-0.5 rounded-full font-mono ${
                typeParam === 'users'
                  ? 'bg-white/20 text-white'
                  : 'bg-forge-surface text-forge-muted border border-forge-border'
              }`}
            >
              {counts.users}
            </span>
          </button>
        </div>

        {/* Right Content Area: Results */}
        <div className="md:col-span-3 space-y-4">
          {/* Results Summary Bar */}
          <div className="flex items-center justify-between pb-2 border-b border-forge-border">
            <div className="text-sm font-medium text-forge-text">
              {queryParam ? (
                <span>
                  <strong>{totalCount}</strong> results for &ldquo;<strong>{queryParam}</strong>&rdquo;
                </span>
              ) : (
                <span>Type a query above to start searching</span>
              )}
            </div>

            {typeParam === 'repositories' && (
              <div className="flex items-center space-x-2 text-xs">
                <span className="text-forge-muted">Sort:</span>
                <select
                  value={sortParam}
                  onChange={e => handleSortChange(e.target.value)}
                  className="bg-forge-card border border-forge-border text-forge-text rounded-md px-2 py-1 focus:outline-none focus:border-forge-accent text-xs"
                >
                  <option value="">Recently Updated</option>
                  <option value="stars">Most Stars</option>
                </select>
              </div>
            )}
          </div>

          {/* Loading State */}
          {isLoading && (
            <div className="py-20 text-center space-y-3">
              <div className="w-8 h-8 border-2 border-forge-accent border-t-transparent rounded-full animate-spin mx-auto"></div>
              <p className="text-sm text-forge-muted">Searching across ForgeHub...</p>
            </div>
          )}

          {/* Error State */}
          {isError && (
            <div className="p-6 bg-rose-500/10 border border-rose-500/20 rounded-xl text-center space-y-2">
              <AlertCircle className="w-8 h-8 text-rose-400 mx-auto" />
              <div className="text-sm font-medium text-rose-300">Failed to load search results</div>
              <p className="text-xs text-rose-400/80">Please check your query or server connection.</p>
            </div>
          )}

          {/* Empty State */}
          {!isLoading && !isError && queryParam && totalCount === 0 && (
            <div className="bg-forge-card border border-forge-border rounded-xl p-12 text-center space-y-4">
              <Search className="w-12 h-12 text-forge-muted/60 mx-auto" />
              <div className="space-y-1">
                <h3 className="text-base font-semibold text-forge-text">
                  No {typeParam} matched your search
                </h3>
                <p className="text-xs text-forge-muted max-w-md mx-auto">
                  Try checking for spelling errors, using more general search keywords, or clearing qualifiers.
                </p>
              </div>
              <div className="pt-2">
                <button
                  onClick={() => {
                    setInputQuery('');
                    setSearchParams(new URLSearchParams());
                  }}
                  className="px-4 py-1.5 text-xs bg-forge-surface border border-forge-border hover:border-forge-subtle text-forge-text rounded-lg transition-colors"
                >
                  Reset Search
                </button>
              </div>
            </div>
          )}

          {/* Empty Prompt when no query */}
          {!isLoading && !queryParam && (
            <div className="bg-forge-card border border-forge-border rounded-xl p-12 text-center space-y-3">
              <FolderGit2 className="w-12 h-12 text-forge-muted/60 mx-auto" />
              <h3 className="text-base font-semibold text-forge-text">Explore ForgeHub</h3>
              <p className="text-xs text-forge-muted max-w-md mx-auto">
                Search through open source repositories, source code files, bug reports, pull requests, and developers.
              </p>
            </div>
          )}

          {/* Results Lists */}
          {!isLoading && !isError && data && (
            <div className="space-y-3">
              {/* Repositories Tab */}
              {typeParam === 'repositories' && data.repositories && (
                <div className="divide-y divide-forge-border/40 bg-forge-card border border-forge-border rounded-xl overflow-hidden">
                  {data.repositories.map(repo => (
                    <div key={repo.id} className="p-4 hover:bg-forge-surface/40 transition-colors">
                      <div className="flex items-start justify-between">
                        <div className="space-y-1">
                          <div className="flex items-center space-x-2">
                            <FolderGit2 className="w-4 h-4 text-forge-accent shrink-0" />
                            <Link
                              to={`/${repo.owner_name}/${repo.slug}`}
                              className="text-base font-semibold text-forge-accent hover:underline"
                            >
                              {repo.owner_name}/{repo.slug}
                            </Link>
                            <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded border border-forge-border bg-forge-bg text-forge-muted">
                              {repo.visibility}
                            </span>
                          </div>
                          {repo.description && (
                            <p className="text-sm text-forge-muted">
                              {repo.description}
                            </p>
                          )}
                          <div className="flex items-center space-x-4 text-xs text-forge-muted pt-2">
                            <span className="flex items-center gap-1">
                              <Star className="w-3.5 h-3.5 text-amber-400" />
                              {repo.star_count}
                            </span>
                            <span className="flex items-center gap-1">
                              <GitFork className="w-3.5 h-3.5" />
                              {repo.fork_count}
                            </span>
                            <span className="flex items-center gap-1">
                              <Clock className="w-3.5 h-3.5" />
                              Updated {new Date(repo.updated_at).toLocaleDateString()}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Code Tab */}
              {typeParam === 'code' && data.code && (
                <div className="space-y-4">
                  {data.code.map((item, idx) => (
                    <div key={idx} className="bg-forge-card border border-forge-border rounded-xl overflow-hidden shadow-sm">
                      {/* Code Header */}
                      <div className="px-4 py-2.5 bg-forge-surface/80 border-b border-forge-border flex items-center justify-between text-xs">
                        <div className="flex items-center space-x-2 font-mono">
                          <CodeIcon className="w-4 h-4 text-amber-400 shrink-0" />
                          <Link
                            to={`/${item.repo_owner}/${item.repo_slug}`}
                            className="text-forge-text hover:text-forge-accent font-medium"
                          >
                            {item.repo_owner}/{item.repo_slug}
                          </Link>
                          <span className="text-forge-muted">&rsaquo;</span>
                          <Link
                            to={`/${item.repo_owner}/${item.repo_slug}/blob/main/${item.file_path}`}
                            className="text-forge-accent hover:underline font-semibold"
                          >
                            {item.file_path}
                          </Link>
                        </div>
                        <span className="text-forge-muted text-[11px]">
                          {item.matches.length} {item.matches.length === 1 ? 'match' : 'matches'}
                        </span>
                      </div>

                      {/* Code Snippets */}
                      <div className="p-3 font-mono text-xs overflow-x-auto divide-y divide-forge-border/20 bg-forge-bg">
                        {item.matches.map((m, mIdx) => (
                          <div
                            key={mIdx}
                            onClick={() => navigate(`/${item.repo_owner}/${item.repo_slug}/blob/main/${item.file_path}`)}
                            className="flex items-center py-1 hover:bg-forge-surface/60 rounded px-2 cursor-pointer transition-colors group"
                          >
                            <span className="w-12 text-forge-subtle group-hover:text-forge-accent select-none shrink-0 text-right pr-4">
                              {m.line_number}
                            </span>
                            <span className="text-forge-text whitespace-pre overflow-hidden text-ellipsis">
                              {m.line_text}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Issues Tab */}
              {typeParam === 'issues' && data.issues && (
                <div className="divide-y divide-forge-border/40 bg-forge-card border border-forge-border rounded-xl overflow-hidden">
                  {data.issues.map(issue => (
                    <div key={issue.id} className="p-4 hover:bg-forge-surface/40 transition-colors">
                      <div className="flex items-start justify-between">
                        <div className="space-y-1">
                          <div className="flex items-center space-x-2">
                            {issue.state === 'open' ? (
                              <AlertCircle className="w-4 h-4 text-emerald-400 shrink-0" />
                            ) : (
                              <CheckCircle2 className="w-4 h-4 text-purple-400 shrink-0" />
                            )}
                            <Link
                              to={`/${issue.repo_owner}/${issue.repo_slug}`}
                              className="text-xs text-forge-muted hover:text-forge-accent"
                            >
                              {issue.repo_owner}/{issue.repo_slug}
                            </Link>
                            <span className="text-xs text-forge-muted font-mono">#{issue.number}</span>
                            <Link
                              to={`/${issue.repo_owner}/${issue.repo_slug}/issues/${issue.number}`}
                              className="text-sm font-semibold text-forge-text hover:text-forge-accent"
                            >
                              {issue.title}
                            </Link>
                          </div>
                          {issue.body_snippet && (
                            <p className="text-xs text-forge-muted line-clamp-2 pl-6">
                              {issue.body_snippet}
                            </p>
                          )}
                          <div className="flex items-center space-x-4 text-xs text-forge-subtle pl-6 pt-1">
                            {issue.author && (
                              <span>opened by <strong>{issue.author.username}</strong></span>
                            )}
                            <span className="flex items-center gap-1">
                              <MessageSquare className="w-3.5 h-3.5" />
                              {issue.comments_count}
                            </span>
                            <span>{new Date(issue.created_at).toLocaleDateString()}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Pull Requests Tab */}
              {typeParam === 'pulls' && data.pulls && (
                <div className="divide-y divide-forge-border/40 bg-forge-card border border-forge-border rounded-xl overflow-hidden">
                  {data.pulls.map(pr => (
                    <div key={pr.id} className="p-4 hover:bg-forge-surface/40 transition-colors">
                      <div className="flex items-start justify-between">
                        <div className="space-y-1">
                          <div className="flex items-center space-x-2">
                            <GitPullRequest className={`w-4 h-4 shrink-0 ${
                              pr.state === 'open' ? 'text-emerald-400' : pr.state === 'merged' ? 'text-purple-400' : 'text-forge-muted'
                            }`} />
                            <Link
                              to={`/${pr.repo_owner}/${pr.repo_slug}`}
                              className="text-xs text-forge-muted hover:text-forge-accent"
                            >
                              {pr.repo_owner}/{pr.repo_slug}
                            </Link>
                            <span className="text-xs text-forge-muted font-mono">#{pr.number}</span>
                            <Link
                              to={`/${pr.repo_owner}/${pr.repo_slug}/pulls/${pr.number}`}
                              className="text-sm font-semibold text-forge-text hover:text-forge-accent"
                            >
                              {pr.title}
                            </Link>
                            {pr.is_draft && (
                              <span className="text-[10px] font-mono uppercase px-1.5 py-0.5 rounded bg-forge-surface text-forge-muted border border-forge-border">
                                Draft
                              </span>
                            )}
                          </div>
                          {pr.body_snippet && (
                            <p className="text-xs text-forge-muted line-clamp-2 pl-6">
                              {pr.body_snippet}
                            </p>
                          )}
                          <div className="flex items-center space-x-4 text-xs text-forge-subtle pl-6 pt-1">
                            <span className="font-mono text-[11px] text-forge-muted">
                              {pr.source_branch} &rarr; {pr.target_branch}
                            </span>
                            {pr.author && (
                              <span>by <strong>{pr.author.username}</strong></span>
                            )}
                            <span>{new Date(pr.created_at).toLocaleDateString()}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Users & Organizations Tab */}
              {typeParam === 'users' && data.users && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  {data.users.map(u => (
                    <Link
                      key={u.id}
                      to={u.is_org ? `/orgs/${u.username}` : `/users/${u.username}`}
                      className="p-4 bg-forge-card border border-forge-border hover:border-forge-subtle rounded-xl flex items-center justify-between group transition-colors shadow-sm"
                    >
                      <div className="flex items-center space-x-3">
                        <div className="w-10 h-10 rounded-full bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center text-white font-semibold text-sm shadow overflow-hidden shrink-0">
                          {u.avatar_url ? (
                            <img src={u.avatar_url} alt={u.username} className="w-full h-full object-cover" />
                          ) : (
                            u.username.substring(0, 2).toUpperCase()
                          )}
                        </div>
                        <div className="space-y-0.5">
                          <div className="text-sm font-semibold text-forge-text group-hover:text-forge-accent flex items-center gap-1.5">
                            <span>{u.username}</span>
                            {u.is_org && (
                              <Building2 className="w-3.5 h-3.5 text-forge-accent" />
                            )}
                          </div>
                          {u.display_name && (
                            <div className="text-xs text-forge-muted">{u.display_name}</div>
                          )}
                          {u.bio && (
                            <div className="text-xs text-forge-subtle line-clamp-1">{u.bio}</div>
                          )}
                        </div>
                      </div>
                      <ArrowRight className="w-4 h-4 text-forge-muted group-hover:text-forge-accent group-hover:translate-x-0.5 transition-all shrink-0" />
                    </Link>
                  ))}
                </div>
              )}

              {/* Pagination Controls */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between pt-4 border-t border-forge-border text-xs text-forge-muted">
                  <button
                    disabled={pageParam <= 1}
                    onClick={() => handlePageChange(pageParam - 1)}
                    className="px-3 py-1.5 rounded-lg bg-forge-card border border-forge-border hover:bg-forge-surface disabled:opacity-40 disabled:pointer-events-none transition-colors"
                  >
                    Previous
                  </button>
                  <span>
                    Page {pageParam} of {totalPages}
                  </span>
                  <button
                    disabled={pageParam >= totalPages}
                    onClick={() => handlePageChange(pageParam + 1)}
                    className="px-3 py-1.5 rounded-lg bg-forge-card border border-forge-border hover:bg-forge-surface disabled:opacity-40 disabled:pointer-events-none transition-colors"
                  >
                    Next
                  </button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
