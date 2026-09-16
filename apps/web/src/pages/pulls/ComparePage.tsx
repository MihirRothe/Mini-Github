import React, { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Repository, DiffResult, GitCommit } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import { DiffViewer } from '../../components/diff/DiffViewer';
import {
  GitPullRequest,
  GitBranch,
  ArrowRight,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  AlertCircle,
  GitCommit as GitCommitIcon
} from 'lucide-react';

interface BranchItem {
  name: string;
  commit_sha: string;
  is_default: boolean;
}

export const ComparePage: React.FC = () => {
  const { owner, repo: repoSlug, spec } = useParams<{ owner: string; repo: string; spec?: string }>();
  const navigate = useNavigate();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [branches, setBranches] = useState<BranchItem[]>([]);
  const [baseBranch, setBaseBranch] = useState<string>('main');
  const [headBranch, setHeadBranch] = useState<string>('main');

  const [diff, setDiff] = useState<DiffResult | null>(null);
  const [commits, setCommits] = useState<GitCommit[]>([]);
  const [canMerge, setCanMerge] = useState<boolean>(true);

  // Form State
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [isDraft, setIsDraft] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const [loadingCompare, setLoadingCompare] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 1. Initial Load: Repository & Branches
  useEffect(() => {
    if (!owner || !repoSlug) return;

    fetch(`/api/v1/repos/${owner}/${repoSlug}`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.repository) {
          setRepo(data.repository);
          const def = data.repository.default_branch || 'main';
          setBaseBranch(def);

          // If spec provided e.g. "main...feature"
          if (spec && spec.includes('...')) {
            const [b, h] = spec.split('...');
            setBaseBranch(b);
            setHeadBranch(h);
          }
        }
      })
      .catch(() => {});

    fetch(`/api/v1/repos/${owner}/${repoSlug}/branches`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.branches) {
          setBranches(data.branches);
          if (!spec) {
            // Pick default branch as base, and another branch as head if available
            const def = data.branches.find((b: BranchItem) => b.is_default)?.name || 'main';
            setBaseBranch(def);
            const other = data.branches.find((b: BranchItem) => b.name !== def);
            if (other) {
              setHeadBranch(other.name);
            } else {
              setHeadBranch(def);
            }
          }
        }
      })
      .catch(() => {});
  }, [owner, repoSlug, spec]);

  // 2. Perform Comparison when base or head changes
  useEffect(() => {
    if (!owner || !repoSlug || !baseBranch || !headBranch) return;
    if (baseBranch === headBranch) {
      setDiff({ files: [], total_files: 0, total_additions: 0, total_deletions: 0 });
      setCommits([]);
      setCanMerge(true);
      return;
    }

    setLoadingCompare(true);
    setError(null);

    fetch(`/api/v1/repos/${owner}/${repoSlug}/compare/${baseBranch}...${headBranch}`, {
      credentials: 'include',
    })
      .then(async (res) => {
        const data = await res.json();
        if (!res.ok) {
          throw new Error(data.error?.message || 'Failed to compare branches');
        }
        return data;
      })
      .then((data) => {
        setDiff(data.diff || { files: [], total_files: 0, total_additions: 0, total_deletions: 0 });
        setCommits(data.commits || []);
        setCanMerge(data.can_merge !== false);

        // Pre-fill title if empty from the last commit message
        if (!title && data.commits && data.commits.length > 0) {
          setTitle(data.commits[0].message.split('\n')[0]);
        }
      })
      .catch((err: any) => {
        setError(err.message || 'Comparison failed');
      })
      .finally(() => {
        setLoadingCompare(false);
      });
  }, [owner, repoSlug, baseBranch, headBranch]);

  // 3. Create Pull Request Submit
  const handleCreatePR = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError('Please provide a title for the pull request.');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          title: title.trim(),
          body: body.trim(),
          source_branch: headBranch,
          target_branch: baseBranch,
          is_draft: isDraft,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to create pull request');
      }

      navigate(`/${owner}/${repoSlug}/pulls/${data.pull_request.number}`);
    } catch (err: any) {
      setError(err.message || 'Error creating pull request');
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-16">
      {repo && <RepoHeader repo={repo} activeTab="pulls" />}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        {/* Title & Branch Selection Toolbar */}
        <div className="card p-4 space-y-4">
          <div className="flex items-center gap-2">
            <GitPullRequest className="w-5 h-5 text-forge-accent" />
            <h1 className="text-base sm:text-lg font-semibold text-white">
              Comparing changes
            </h1>
          </div>

          <p className="text-xs text-forge-muted">
            Choose two branches to see what's changed or to start a new pull request.
          </p>

          <div className="flex flex-wrap items-center gap-3 pt-1">
            {/* Base Branch Selector */}
            <div className="flex items-center gap-2 bg-forge-bg px-3 py-1.5 rounded-lg border border-forge-border">
              <span className="text-xs text-forge-muted font-medium">base:</span>
              <GitBranch className="w-3.5 h-3.5 text-forge-accent" />
              <select
                value={baseBranch}
                onChange={(e) => setBaseBranch(e.target.value)}
                className="bg-transparent text-xs text-white font-mono focus:outline-none cursor-pointer"
              >
                {branches.map((b) => (
                  <option key={b.name} value={b.name} className="bg-forge-card text-white">
                    {b.name}
                  </option>
                ))}
              </select>
            </div>

            <ArrowRight className="w-4 h-4 text-forge-muted shrink-0" />

            {/* Compare/Head Branch Selector */}
            <div className="flex items-center gap-2 bg-forge-bg px-3 py-1.5 rounded-lg border border-forge-border">
              <span className="text-xs text-forge-muted font-medium">compare:</span>
              <GitBranch className="w-3.5 h-3.5 text-purple-400" />
              <select
                value={headBranch}
                onChange={(e) => setHeadBranch(e.target.value)}
                className="bg-transparent text-xs text-white font-mono focus:outline-none cursor-pointer"
              >
                {branches.map((b) => (
                  <option key={b.name} value={b.name} className="bg-forge-card text-white">
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Mergeability Status Banner */}
          {baseBranch !== headBranch && !loadingCompare && (
            <div className="pt-2">
              {canMerge ? (
                <div className="p-3 bg-emerald-500/10 border border-emerald-500/30 rounded-lg flex items-center gap-2.5 text-xs text-emerald-300 font-medium">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                  <span>
                    Able to merge. These branches can be automatically merged.
                  </span>
                </div>
              ) : (
                <div className="p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg flex items-center gap-2.5 text-xs text-amber-300 font-medium">
                  <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0" />
                  <span>
                    Can't automatically merge. Don't worry, you can still create the pull request.
                  </span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
            <AlertCircle className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Comparison Loading */}
        {loadingCompare ? (
          <div className="card p-12 text-center text-forge-muted">
            <Loader2 className="w-6 h-6 text-forge-accent animate-spin mx-auto mb-2" />
            <p className="text-xs">Computing diff between {baseBranch} and {headBranch}...</p>
          </div>
        ) : baseBranch === headBranch ? (
          <div className="card p-12 text-center text-forge-muted space-y-2">
            <GitBranch className="w-8 h-8 text-forge-muted/40 mx-auto" />
            <p className="text-sm font-medium text-white">There isn't anything to compare.</p>
            <p className="text-xs text-forge-muted">
              You selected the same branch for both base and compare ({baseBranch}).
            </p>
          </div>
        ) : (
          <div className="space-y-6">
            {/* PR Creation Box */}
            <div className="card p-6 space-y-4 border border-forge-border">
              <h2 className="text-sm font-semibold text-white">Open a pull request</h2>
              <form onSubmit={handleCreatePR} className="space-y-4">
                <div>
                  <label className="block text-xs font-medium text-forge-muted mb-1">
                    Title <span className="text-rose-400">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="Title"
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="w-full px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-sm text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent transition"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-forge-muted mb-1">
                    Description
                  </label>
                  <textarea
                    rows={4}
                    placeholder="Leave a comment describing the changes in this pull request..."
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    className="w-full px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-sm text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent transition font-mono"
                  />
                </div>

                <div className="flex items-center justify-between pt-2">
                  <label className="flex items-center gap-2 cursor-pointer text-xs text-forge-muted select-none">
                    <input
                      type="checkbox"
                      checked={isDraft}
                      onChange={(e) => setIsDraft(e.target.checked)}
                      className="rounded border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                    />
                    <span>Create as draft</span>
                  </label>

                  <button
                    type="submit"
                    disabled={submitting || !title.trim()}
                    className="btn-primary text-xs flex items-center gap-2 disabled:opacity-50"
                  >
                    {submitting && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                    <span>Create pull request</span>
                  </button>
                </div>
              </form>
            </div>

            {/* Commits List */}
            {commits && commits.length > 0 && (
              <div className="card overflow-hidden border border-forge-border">
                <div className="px-4 py-3 bg-forge-card/80 border-b border-forge-border flex items-center gap-2 text-xs font-semibold text-white">
                  <GitCommitIcon className="w-4 h-4 text-forge-accent" />
                  <span>Commits ({commits.length})</span>
                </div>
                <div className="divide-y divide-forge-border/40 text-xs">
                  {commits.map((c) => (
                    <div key={c.sha} className="p-3 hover:bg-forge-card/40 flex items-center justify-between gap-4">
                      <div className="space-y-0.5 min-w-0">
                        <p className="font-medium text-white truncate">{c.message.split('\n')[0]}</p>
                        <p className="text-[11px] text-forge-muted">
                          {c.author_name} committed on {c.date}
                        </p>
                      </div>
                      <span className="font-mono text-[11px] px-2 py-0.5 bg-forge-bg rounded border border-forge-border text-forge-accent shrink-0">
                        {c.short_sha || c.sha.slice(0, 7)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Visual Diff View */}
            {diff && <DiffViewer diff={diff} />}
          </div>
        )}
      </div>
    </div>
  );
};
