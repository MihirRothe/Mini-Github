import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, Issue, Label, Milestone } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import {
  CircleDot,
  CheckCircle2,
  Milestone as MilestoneIcon,
  Search,
  Plus,
  MessageSquare,
  Loader2,
  AlertCircle
} from 'lucide-react';

export const IssuesListPage: React.FC = () => {
  const { owner, repo: repoSlug } = useParams<{ owner: string; repo: string }>();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [issues, setIssues] = useState<Issue[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);
  const [milestones, setMilestones] = useState<Milestone[]>([]);
  const [openCount, setOpenCount] = useState<number>(0);
  const [closedCount, setClosedCount] = useState<number>(0);

  const [stateFilter, setStateFilter] = useState<'open' | 'closed'>('open');
  const [selectedLabel, setSelectedLabel] = useState<string>('');
  const [selectedMilestone, setSelectedMilestone] = useState<string>('');
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

    // Fetch Labels & Milestones for filter dropdowns
    fetch(`/api/v1/repos/${owner}/${repoSlug}/labels`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => setLabels(data?.labels || []))
      .catch(() => {});

    fetch(`/api/v1/repos/${owner}/${repoSlug}/milestones`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => setMilestones(data?.milestones || []))
      .catch(() => {});
  }, [owner, repoSlug]);

  // 2. Fetch Issues with filters
  const fetchIssues = async () => {
    if (!owner || !repoSlug) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    params.set('state', stateFilter);
    if (selectedLabel) params.set('label', selectedLabel);
    if (selectedMilestone) params.set('milestone', selectedMilestone);
    if (searchQuery.trim()) params.set('q', searchQuery.trim());

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/issues?${params.toString()}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load issues.');
      }
      setIssues(data.issues || []);
      setOpenCount(data.open_count || 0);
      setClosedCount(data.closed_count || 0);
    } catch (err: any) {
      setError(err.message || 'Error fetching issues.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchIssues();
  }, [owner, repoSlug, stateFilter, selectedLabel, selectedMilestone]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    fetchIssues();
  };

  if (!repo && loading) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading repository...</p>
      </div>
    );
  }

  if (!repo) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <AlertCircle className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Repository Not Found</h2>
        <Link to="/" className="btn-secondary">Return Home</Link>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-forge-bg pb-16">
      <RepoHeader repo={repo} activeTab="issues" openIssuesCount={openCount} />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-4">
        {/* Controls Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
          {/* Search bar */}
          <form onSubmit={handleSearchSubmit} className="flex-1 max-w-md relative">
            <Search className="w-4 h-4 text-forge-muted absolute left-3 top-2.5" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search all issues..."
              className="w-full pl-9 pr-4 py-2 bg-forge-card border border-forge-border rounded-lg text-xs text-white placeholder:text-forge-muted focus:outline-none focus:border-blue-500 transition-colors"
            />
          </form>

          {/* Filters & New Issue Button */}
          <div className="flex flex-wrap items-center gap-2">
            {/* Label filter */}
            {labels.length > 0 && (
              <div className="relative">
                <select
                  value={selectedLabel}
                  onChange={(e) => setSelectedLabel(e.target.value)}
                  className="bg-forge-card border border-forge-border rounded-lg px-2.5 py-1.5 text-xs text-white focus:outline-none cursor-pointer"
                >
                  <option value="">Label: All</option>
                  {labels.map((l) => (
                    <option key={l.id} value={l.name}>
                      {l.name}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* Milestone filter */}
            {milestones.length > 0 && (
              <div className="relative">
                <select
                  value={selectedMilestone}
                  onChange={(e) => setSelectedMilestone(e.target.value)}
                  className="bg-forge-card border border-forge-border rounded-lg px-2.5 py-1.5 text-xs text-white focus:outline-none cursor-pointer"
                >
                  <option value="">Milestone: All</option>
                  {milestones.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.title}
                    </option>
                  ))}
                </select>
              </div>
            )}

            <Link
              to={`/${owner}/${repoSlug}/issues/new`}
              className="btn-primary text-xs flex items-center gap-1.5 shadow-lg shadow-blue-600/20"
            >
              <Plus className="w-4 h-4" />
              <span>New Issue</span>
            </Link>
          </div>
        </div>

        {/* Issues List Card */}
        <div className="card overflow-hidden">
          {/* Card Header: Open / Closed tabs */}
          <div className="p-3 px-4 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs">
            <div className="flex items-center gap-4">
              <button
                onClick={() => setStateFilter('open')}
                className={`flex items-center gap-1.5 font-semibold transition-colors ${
                  stateFilter === 'open' ? 'text-white' : 'text-forge-muted hover:text-white'
                }`}
              >
                <CircleDot className="w-4 h-4 text-emerald-400" />
                <span>{openCount} Open</span>
              </button>

              <button
                onClick={() => setStateFilter('closed')}
                className={`flex items-center gap-1.5 font-semibold transition-colors ${
                  stateFilter === 'closed' ? 'text-white' : 'text-forge-muted hover:text-white'
                }`}
              >
                <CheckCircle2 className="w-4 h-4 text-purple-400" />
                <span>{closedCount} Closed</span>
              </button>
            </div>

            {(selectedLabel || selectedMilestone || searchQuery) && (
              <button
                onClick={() => {
                  setSelectedLabel('');
                  setSelectedMilestone('');
                  setSearchQuery('');
                }}
                className="text-forge-accent hover:underline text-[11px]"
              >
                Clear filters
              </button>
            )}
          </div>

          {/* List Content */}
          {loading ? (
            <div className="p-12 text-center space-y-2">
              <Loader2 className="w-6 h-6 text-forge-accent animate-spin mx-auto" />
              <p className="text-xs text-forge-muted">Loading issues...</p>
            </div>
          ) : error ? (
            <div className="p-8 text-center text-rose-400 text-xs">{error}</div>
          ) : issues.length === 0 ? (
            <div className="p-16 text-center space-y-3">
              <CircleDot className="w-12 h-12 text-forge-muted/40 mx-auto" />
              <h3 className="text-sm font-semibold text-white">
                {stateFilter === 'open' ? 'No open issues' : 'No closed issues'}
              </h3>
              <p className="text-xs text-forge-muted max-w-sm mx-auto">
                {selectedLabel || selectedMilestone || searchQuery
                  ? 'No issues match the selected filters.'
                  : 'Get started by creating a new issue to track bugs or tasks.'}
              </p>
              <div className="pt-2">
                <Link
                  to={`/${owner}/${repoSlug}/issues/new`}
                  className="btn-primary text-xs inline-flex items-center gap-1.5"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Create Issue</span>
                </Link>
              </div>
            </div>
          ) : (
            <div className="divide-y divide-forge-border">
              {issues.map((issue) => (
                <div
                  key={issue.id}
                  className="p-4 flex items-start justify-between gap-4 hover:bg-forge-card/40 transition-colors"
                >
                  <div className="flex items-start gap-3 min-w-0">
                    {/* Status Icon */}
                    {issue.state === 'open' ? (
                      <CircleDot className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                    ) : (
                      <CheckCircle2 className="w-4 h-4 text-purple-400 shrink-0 mt-0.5" />
                    )}

                    {/* Title and metadata */}
                    <div className="space-y-1 min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <Link
                          to={`/${owner}/${repoSlug}/issues/${issue.number}`}
                          className="font-semibold text-sm text-white hover:text-blue-400 transition-colors"
                        >
                          {issue.title}
                        </Link>

                        {/* Label badges */}
                        {issue.labels.map((l) => (
                          <span
                            key={l.id}
                            style={{
                              backgroundColor: `${l.color}25`,
                              borderColor: `${l.color}60`,
                              color: l.color,
                            }}
                            className="text-[10px] font-medium px-2 py-0.2 rounded-full border"
                          >
                            {l.name}
                          </span>
                        ))}
                      </div>

                      {/* Sub-line: #N opened time by user · milestone */}
                      <div className="flex flex-wrap items-center gap-2 text-xs text-forge-muted">
                        <span>#{issue.number}</span>
                        <span>
                          {issue.state === 'open' ? 'opened' : 'closed'}{' '}
                          {new Date(issue.created_at).toLocaleDateString()}
                        </span>
                        {issue.author && (
                          <span>
                            by <span className="text-forge-text font-medium">{issue.author.username}</span>
                          </span>
                        )}

                        {issue.milestone && (
                          <span className="flex items-center gap-1 text-[11px] text-forge-text">
                            <MilestoneIcon className="w-3 h-3 text-forge-muted" />
                            {issue.milestone.title}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Right side: Comments count & Assignees */}
                  <div className="flex items-center gap-4 shrink-0 self-center">
                    {issue.assignees.length > 0 && (
                      <div className="flex -space-x-1.5 overflow-hidden">
                        {issue.assignees.map((u) => (
                          <div
                            key={u.id}
                            title={u.username}
                            className="w-5 h-5 rounded-full bg-gradient-to-tr from-blue-600 to-indigo-600 border border-forge-bg flex items-center justify-center text-[9px] font-bold text-white uppercase select-none"
                          >
                            {u.username.slice(0, 1)}
                          </div>
                        ))}
                      </div>
                    )}

                    {issue.comments_count > 0 && (
                      <Link
                        to={`/${owner}/${repoSlug}/issues/${issue.number}`}
                        className="flex items-center gap-1 text-xs text-forge-muted hover:text-white transition-colors"
                      >
                        <MessageSquare className="w-3.5 h-3.5" />
                        <span>{issue.comments_count}</span>
                      </Link>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
