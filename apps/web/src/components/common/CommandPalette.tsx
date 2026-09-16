import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Search,
  FolderGit2,
  Activity,
  Compass,
  PlusCircle,
  BookOpen,
  X,
  AlertCircle,
  GitPullRequest,
  Code,
  User,
  Building2,
  ArrowRight,
  Loader2
} from 'lucide-react';
import { QuickSearchResponse } from '../../types';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
}

export const CommandPalette: React.FC<CommandPaletteProps> = ({ isOpen, onClose }) => {
  const [query, setQuery] = useState('');
  const [quickResults, setQuickResults] = useState<QuickSearchResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        isOpen ? onClose() : onClose();
      }
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  // Debounced search query
  useEffect(() => {
    if (!query.trim()) {
      setQuickResults(null);
      setIsLoading(false);
      return;
    }

    const timer = setTimeout(async () => {
      setIsLoading(true);
      try {
        const res = await fetch(`/api/v1/search/quick?q=${encodeURIComponent(query)}`);
        if (res.ok) {
          const data = await res.json();
          setQuickResults(data);
        }
      } catch (err) {
        console.error('Quick search error:', err);
      } finally {
        setIsLoading(false);
      }
    }, 250);

    return () => clearTimeout(timer);
  }, [query]);

  if (!isOpen) return null;

  const defaultItems = [
    { label: 'Go to Dashboard', icon: FolderGit2, action: () => navigate('/dashboard'), category: 'Navigation' },
    { label: 'Explore Repositories', icon: Compass, action: () => navigate('/explore'), category: 'Navigation' },
    { label: 'System Health & Telemetry', icon: Activity, action: () => navigate('/status'), category: 'Diagnostics' },
    { label: 'Create New Repository', icon: PlusCircle, action: () => navigate('/dashboard'), category: 'Actions' },
    { label: 'Read Architecture Docs', icon: BookOpen, action: () => window.open('https://github.com', '_blank'), category: 'Help' },
  ];

  const handleFullSearch = () => {
    if (query.trim()) {
      navigate(`/search?q=${encodeURIComponent(query.trim())}`);
      onClose();
    }
  };

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleFullSearch();
    }
  };

  const hasResults = quickResults && (
    quickResults.repositories.length > 0 ||
    quickResults.issues.length > 0 ||
    quickResults.pulls.length > 0 ||
    quickResults.code.length > 0 ||
    quickResults.users.length > 0
  );

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 bg-black/75 backdrop-blur-sm p-4 animate-fade-in">
      <div className="w-full max-w-2xl bg-forge-surface border border-forge-border rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">
        {/* Search Input Header */}
        <div className="flex items-center px-4 py-3 border-b border-forge-border">
          {isLoading ? (
            <Loader2 className="w-5 h-5 text-forge-accent animate-spin mr-3 shrink-0" />
          ) : (
            <Search className="w-5 h-5 text-forge-muted mr-3 shrink-0" />
          )}
          <input
            autoFocus
            type="text"
            placeholder="Search repositories, issues, PRs, code, users... (or type a command)"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleInputKeyDown}
            className="w-full bg-transparent text-forge-text text-sm focus:outline-none placeholder:text-forge-muted"
          />
          {query && (
            <button
              onClick={() => setQuery('')}
              className="text-forge-muted hover:text-forge-text p-1 mr-1 text-xs"
            >
              Clear
            </button>
          )}
          <button
            onClick={onClose}
            className="text-forge-muted hover:text-forge-text p-1 rounded-md"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-3 space-y-4">
          {/* Query prompt to full search page */}
          {query.trim() && (
            <button
              onClick={handleFullSearch}
              className="w-full flex items-center justify-between px-3 py-2 rounded-lg bg-forge-accent/10 border border-forge-accent/20 hover:bg-forge-accent/20 text-forge-accent text-sm transition-colors text-left font-medium"
            >
              <div className="flex items-center space-x-2">
                <Search className="w-4 h-4" />
                <span>See all search results for &ldquo;<strong>{query}</strong>&rdquo;</span>
              </div>
              <div className="flex items-center space-x-1 text-xs opacity-80">
                <span>Press Enter</span>
                <ArrowRight className="w-3 h-3" />
              </div>
            </button>
          )}

          {/* Quick results if typing */}
          {query.trim() && quickResults && (
            <>
              {/* Repositories */}
              {quickResults.repositories.length > 0 && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted px-3 py-1">
                    Repositories
                  </div>
                  <div className="space-y-1">
                    {quickResults.repositories.map(repo => (
                      <button
                        key={repo.id}
                        onClick={() => {
                          navigate(`/${repo.owner_name}/${repo.slug}`);
                          onClose();
                        }}
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-forge-text hover:bg-forge-card transition-colors text-left group"
                      >
                        <div className="flex items-center space-x-2.5 overflow-hidden">
                          <FolderGit2 className="w-4 h-4 text-forge-accent shrink-0" />
                          <span className="font-medium group-hover:text-forge-accent truncate">
                            {repo.owner_name}/{repo.slug}
                          </span>
                          {repo.description && (
                            <span className="text-xs text-forge-muted truncate hidden sm:inline">
                              — {repo.description}
                            </span>
                          )}
                        </div>
                        <span className="text-xs text-forge-muted shrink-0 ml-2">
                          {repo.visibility}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Issues */}
              {quickResults.issues.length > 0 && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted px-3 py-1">
                    Issues
                  </div>
                  <div className="space-y-1">
                    {quickResults.issues.map(issue => (
                      <button
                        key={issue.id}
                        onClick={() => {
                          navigate(`/${issue.repo_owner}/${issue.repo_slug}/issues/${issue.number}`);
                          onClose();
                        }}
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-forge-text hover:bg-forge-card transition-colors text-left group"
                      >
                        <div className="flex items-center space-x-2.5 overflow-hidden">
                          <AlertCircle className={`w-4 h-4 shrink-0 ${issue.state === 'open' ? 'text-emerald-400' : 'text-purple-400'}`} />
                          <span className="text-forge-muted font-mono text-xs">#{issue.number}</span>
                          <span className="font-medium group-hover:text-forge-accent truncate">
                            {issue.title}
                          </span>
                        </div>
                        <span className="text-xs text-forge-muted shrink-0 ml-2">
                          {issue.repo_owner}/{issue.repo_slug}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Pull Requests */}
              {quickResults.pulls.length > 0 && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted px-3 py-1">
                    Pull Requests
                  </div>
                  <div className="space-y-1">
                    {quickResults.pulls.map(pr => (
                      <button
                        key={pr.id}
                        onClick={() => {
                          navigate(`/${pr.repo_owner}/${pr.repo_slug}/pulls/${pr.number}`);
                          onClose();
                        }}
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-forge-text hover:bg-forge-card transition-colors text-left group"
                      >
                        <div className="flex items-center space-x-2.5 overflow-hidden">
                          <GitPullRequest className="w-4 h-4 text-purple-400 shrink-0" />
                          <span className="text-forge-muted font-mono text-xs">#{pr.number}</span>
                          <span className="font-medium group-hover:text-forge-accent truncate">
                            {pr.title}
                          </span>
                        </div>
                        <span className="text-xs text-forge-muted shrink-0 ml-2">
                          {pr.repo_owner}/{pr.repo_slug}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Code */}
              {quickResults.code.length > 0 && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted px-3 py-1">
                    Code Matches
                  </div>
                  <div className="space-y-1">
                    {quickResults.code.map((c, idx) => (
                      <button
                        key={idx}
                        onClick={() => {
                          navigate(`/${c.repo_owner}/${c.repo_slug}/blob/main/${c.file_path}`);
                          onClose();
                        }}
                        className="w-full flex flex-col px-3 py-2 rounded-lg text-sm text-forge-text hover:bg-forge-card transition-colors text-left group space-y-1"
                      >
                        <div className="flex items-center space-x-2 text-xs">
                          <Code className="w-3.5 h-3.5 text-amber-400 shrink-0" />
                          <span className="text-forge-muted">{c.repo_owner}/{c.repo_slug} &rsaquo;</span>
                          <span className="font-mono text-forge-accent font-medium">{c.file_path}</span>
                        </div>
                        {c.matches.length > 0 && (
                          <div className="font-mono text-xs text-forge-muted bg-forge-bg px-2 py-1 rounded border border-forge-border truncate">
                            <span className="text-forge-subtle mr-2">{c.matches[0].line_number}:</span>
                            {c.matches[0].line_text}
                          </div>
                        )}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Users & Orgs */}
              {quickResults.users.length > 0 && (
                <div>
                  <div className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted px-3 py-1">
                    Users &amp; Organizations
                  </div>
                  <div className="space-y-1">
                    {quickResults.users.map(u => (
                      <button
                        key={u.id}
                        onClick={() => {
                          navigate(u.is_org ? `/orgs/${u.username}` : `/users/${u.username}`);
                          onClose();
                        }}
                        className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-forge-text hover:bg-forge-card transition-colors text-left group"
                      >
                        <div className="flex items-center space-x-2.5">
                          {u.is_org ? (
                            <Building2 className="w-4 h-4 text-forge-accent" />
                          ) : (
                            <User className="w-4 h-4 text-emerald-400" />
                          )}
                          <span className="font-medium group-hover:text-forge-accent">
                            {u.username}
                          </span>
                          {u.display_name && (
                            <span className="text-xs text-forge-muted">
                              ({u.display_name})
                            </span>
                          )}
                        </div>
                        <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-forge-bg text-forge-muted border border-forge-border">
                          {u.is_org ? 'Organization' : 'User'}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {!hasResults && !isLoading && (
                <div className="py-8 text-center text-sm text-forge-muted">
                  No quick matches found. Hit Enter to search everywhere with qualifiers.
                </div>
              )}
            </>
          )}

          {/* Default items when query is empty */}
          {!query.trim() && (
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wider text-forge-muted px-3 py-1">
                Quick Actions
              </div>
              <div className="space-y-1">
                {defaultItems.map((item, idx) => {
                  const Icon = item.icon;
                  return (
                    <button
                      key={idx}
                      onClick={() => {
                        item.action();
                        onClose();
                      }}
                      className="w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm text-forge-text hover:bg-forge-card hover:text-forge-accent transition-colors text-left"
                    >
                      <div className="flex items-center space-x-3">
                        <Icon className="w-4 h-4 text-forge-muted" />
                        <span>{item.label}</span>
                      </div>
                      <span className="text-xs text-forge-muted bg-forge-bg px-2 py-0.5 rounded border border-forge-border">
                        {item.category}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="px-4 py-2 bg-forge-bg border-t border-forge-border text-xs text-forge-muted flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <span><kbd className="px-1 py-0.5 bg-forge-surface border border-forge-border rounded text-[10px]">Enter</kbd> to search</span>
            <span><kbd className="px-1 py-0.5 bg-forge-surface border border-forge-border rounded text-[10px]">ESC</kbd> to close</span>
          </div>
          <span className="text-forge-subtle text-[11px]">Tip: Use &quot;repo:&quot;, &quot;is:open&quot;, &quot;author:&quot;</span>
        </div>
      </div>
    </div>
  );
};
