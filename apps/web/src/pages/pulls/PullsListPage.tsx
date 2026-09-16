import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, PullRequest } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import {
  GitPullRequest,
  GitMerge,
  Search,
  Plus,
  MessageSquare,
  Loader2,
  AlertCircle,
  GitBranch,
  CheckCircle2,
  XCircle
} from 'lucide-react';

export const PullsListPage: React.FC = () => {
  const { owner, repo: repoSlug } = useParams<{ owner: string; repo: string }>();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [pulls, setPulls] = useState<PullRequest[]>([]);
  const [openCount, setOpenCount] = useState<number>(0);
  const [closedCount, setClosedCount] = useState<number>(0);
  const [mergedCount, setMergedCount] = useState<number>(0);

  const [stateFilter, setStateFilter] = useState<'open' | 'closed' | 'merged'>('open');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 1. Fetch Repository Details
  useEffect(() => {
    if (!owner || !repoSlug) return;
    fetch(`/api/v1/repos/${owner}/${repoSlug}`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.repository) {
          setRepo(data.repository);
        }
      })
      .catch(() => {});
  }, [owner, repoSlug]);

  // 2. Fetch Pull Requests
  const fetchPulls = async () => {
    if (!owner || !repoSlug) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    params.set('state', stateFilter);
    if (searchQuery.trim()) params.set('q', searchQuery.trim());

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls?${params.toString()}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load pull requests.');
      }
      setPulls(data.pull_requests || []);
      setOpenCount(data.open_count || 0);
      setClosedCount(data.closed_count || 0);
      setMergedCount(data.merged_count || 0);
    } catch (err: any) {
      setError(err.message || 'Error fetching pull requests.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPulls();
  }, [owner, repoSlug, stateFilter]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    fetchPulls();
  };

  if (!repo && loading) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading repository pull requests...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-12">
      {repo && (
        <RepoHeader
          repo={repo}
          activeTab="pulls"
          openPullsCount={openCount}
        />
      )}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        {/* Action Header & Filter Controls */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4">
          <form onSubmit={handleSearchSubmit} className="relative flex-1 max-w-lg">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-forge-muted" />
            <input
              type="text"
              placeholder="Filter pull requests by title or branch..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-4 py-2 bg-forge-card border border-forge-border rounded-lg text-sm text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent transition"
            />
          </form>

          <div className="flex items-center gap-3">
            <Link
              to={`/${owner}/${repoSlug}/compare`}
              className="btn-primary text-xs flex items-center gap-1.5 shadow-lg shadow-forge-accent/10 whitespace-nowrap"
            >
              <Plus className="w-4 h-4" />
              <span>New Pull Request</span>
            </Link>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
            <AlertCircle className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Pull Requests List Card */}
        <div className="card overflow-hidden border border-forge-border">
          {/* Filter Bar */}
          <div className="p-3.5 bg-forge-card/80 border-b border-forge-border flex flex-wrap items-center justify-between gap-4 text-xs select-none">
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setStateFilter('open')}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition ${
                  stateFilter === 'open'
                    ? 'bg-forge-accent/20 text-white border border-forge-accent/30'
                    : 'text-forge-muted hover:text-white hover:bg-forge-card'
                }`}
              >
                <GitPullRequest className="w-4 h-4 text-emerald-400" />
                <span>{openCount} Open</span>
              </button>

              <button
                type="button"
                onClick={() => setStateFilter('merged')}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition ${
                  stateFilter === 'merged'
                    ? 'bg-purple-500/20 text-white border border-purple-500/30'
                    : 'text-forge-muted hover:text-white hover:bg-forge-card'
                }`}
              >
                <GitMerge className="w-4 h-4 text-purple-400" />
                <span>{mergedCount} Merged</span>
              </button>

              <button
                type="button"
                onClick={() => setStateFilter('closed')}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition ${
                  stateFilter === 'closed'
                    ? 'bg-forge-border text-white border border-forge-border'
                    : 'text-forge-muted hover:text-white hover:bg-forge-card'
                }`}
              >
                <XCircle className="w-4 h-4 text-rose-400" />
                <span>{closedCount} Closed</span>
              </button>
            </div>
          </div>

          {/* List Content */}
          {loading ? (
            <div className="p-12 text-center text-forge-muted">
              <Loader2 className="w-6 h-6 text-forge-accent animate-spin mx-auto mb-2" />
              <p className="text-sm">Fetching pull requests...</p>
            </div>
          ) : pulls.length === 0 ? (
            <div className="p-12 text-center text-forge-muted space-y-3">
              <GitPullRequest className="w-10 h-10 mx-auto text-forge-muted/40" />
              <div className="space-y-1">
                <p className="text-white font-medium text-base">No {stateFilter} pull requests</p>
                <p className="text-xs text-forge-muted max-w-sm mx-auto">
                  {stateFilter === 'open'
                    ? 'There are currently no open pull requests. Push branches to your repo and open a pull request to merge changes.'
                    : `No ${stateFilter} pull requests found matching your current filter.`}
                </p>
              </div>
              {stateFilter === 'open' && (
                <Link
                  to={`/${owner}/${repoSlug}/compare`}
                  className="btn-primary text-xs inline-flex items-center gap-1.5 mt-2"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Compare & Open Pull Request
                </Link>
              )}
            </div>
          ) : (
            <div className="divide-y divide-forge-border/60">
              {pulls.map((pr) => {
                const isMerged = pr.state === 'merged';
                const isClosed = pr.state === 'closed';

                return (
                  <div
                    key={pr.id}
                    className="p-4 hover:bg-forge-card/40 transition flex items-start justify-between gap-4"
                  >
                    <div className="flex items-start gap-3 min-w-0">
                      <div className="mt-1 shrink-0">
                        {isMerged ? (
                          <GitMerge className="w-4 h-4 text-purple-400" />
                        ) : isClosed ? (
                          <XCircle className="w-4 h-4 text-rose-400" />
                        ) : (
                          <GitPullRequest className="w-4 h-4 text-emerald-400" />
                        )}
                      </div>

                      <div className="space-y-1.5 min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <Link
                            to={`/${owner}/${repoSlug}/pulls/${pr.number}`}
                            className="font-medium text-white hover:text-forge-accent text-sm sm:text-base leading-snug transition break-words"
                          >
                            {pr.title}
                          </Link>

                          {pr.is_draft && (
                            <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-forge-bg text-forge-muted border border-forge-border">
                              Draft
                            </span>
                          )}

                          {isMerged && (
                            <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-purple-500/10 text-purple-400 border border-purple-500/30 flex items-center gap-1">
                              <CheckCircle2 className="w-3 h-3" /> Merged
                            </span>
                          )}
                        </div>

                        <div className="flex flex-wrap items-center gap-2 text-xs text-forge-muted">
                          <span>#{pr.number}</span>
                          <span>•</span>
                          <span>
                            opened by{' '}
                            <span className="text-forge-muted/90 font-medium">
                              {pr.author?.username || 'user'}
                            </span>
                          </span>
                          <span>•</span>
                          <span className="inline-flex items-center gap-1 font-mono text-[11px] bg-forge-bg px-2 py-0.5 rounded border border-forge-border/80 text-forge-accent">
                            <GitBranch className="w-3 h-3 text-forge-muted" />
                            {pr.source_branch} → {pr.target_branch}
                          </span>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-4 shrink-0 text-forge-muted text-xs">
                      {pr.comments_count > 0 && (
                        <div className="flex items-center gap-1 hover:text-white transition">
                          <MessageSquare className="w-3.5 h-3.5" />
                          <span>{pr.comments_count}</span>
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
