import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, PullRequestDetail, GitCommit } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import { DiffViewer } from '../../components/diff/DiffViewer';
import { useAuth } from '../../context/AuthContext';
import {
  GitPullRequest,
  GitMerge,
  CheckCircle2,
  AlertTriangle,
  MessageSquare,
  GitCommit as GitCommitIcon,
  FileCode,
  Loader2,
  AlertCircle,
  XCircle,
  Check,
  Send,
  Trash2,
  ShieldAlert,
  Sparkles
} from 'lucide-react';
import { AIReviewModal } from '../../components/ai/AIReviewModal';

export const PullDetailPage: React.FC = () => {
  const { owner, repo: repoSlug, number: numberStr } = useParams<{
    owner: string;
    repo: string;
    number: string;
  }>();
  const { user: currentUser } = useAuth();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [pr, setPR] = useState<PullRequestDetail | null>(null);
  const [activeTab, setActiveTab] = useState<'conversation' | 'commits' | 'files'>('conversation');
  const [aiReviewOpen, setAiReviewOpen] = useState(false);

  // New Comment state
  const [commentBody, setCommentBody] = useState('');
  const [submittingComment, setSubmittingComment] = useState(false);

  // New Review state
  const [showReviewBox, setShowReviewBox] = useState(false);
  const [reviewState, setReviewState] = useState<'APPROVED' | 'CHANGES_REQUESTED' | 'COMMENTED'>('APPROVED');
  const [reviewBody, setReviewBody] = useState('');
  const [submittingReview, setSubmittingReview] = useState(false);

  // Merge state
  const [mergeMethod, setMergeMethod] = useState<'merge' | 'squash'>('merge');
  const [commitMessage, setCommitMessage] = useState('');
  const [merging, setMerging] = useState(false);

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

  // 2. Fetch Pull Request Detail
  const fetchPR = async () => {
    if (!owner || !repoSlug || !numberStr) return;
    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls/${numberStr}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load pull request.');
      }
      setPR(data.pull_request);
      setCommitMessage(`Merge pull request #${numberStr} from ${data.pull_request.source_branch}`);
    } catch (err: any) {
      setError(err.message || 'Error fetching pull request');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPR();
  }, [owner, repoSlug, numberStr]);

  // 3. Submit Comment
  const handleAddComment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!commentBody.trim() || !owner || !repoSlug || !numberStr) return;

    setSubmittingComment(true);
    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls/${numberStr}/comments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ body: commentBody.trim() }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error?.message || 'Failed to post comment');
      }
      setCommentBody('');
      await fetchPR();
    } catch (err: any) {
      setError(err.message || 'Failed to post comment');
    } finally {
      setSubmittingComment(false);
    }
  };

  // 4. Submit Review
  const handleSubmitReview = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!owner || !repoSlug || !numberStr) return;

    setSubmittingReview(true);
    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls/${numberStr}/reviews`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          state: reviewState,
          body: reviewBody.trim(),
        }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error?.message || 'Failed to submit review');
      }
      setReviewBody('');
      setShowReviewBox(false);
      await fetchPR();
    } catch (err: any) {
      setError(err.message || 'Failed to submit review');
    } finally {
      setSubmittingReview(false);
    }
  };

  // 5. Delete Comment
  const handleDeleteComment = async (commentId: string) => {
    if (!owner || !repoSlug || !numberStr) return;
    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repoSlug}/pulls/${numberStr}/comments/${commentId}`,
        {
          method: 'DELETE',
          credentials: 'include',
        }
      );
      if (res.ok) {
        await fetchPR();
      }
    } catch (err) {}
  };

  // 6. Merge Pull Request
  const handleMerge = async () => {
    if (!owner || !repoSlug || !numberStr) return;

    setMerging(true);
    setError(null);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls/${numberStr}/merge`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          method: mergeMethod,
          commit_message: commitMessage.trim(),
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to merge pull request');
      }
      await fetchPR();
    } catch (err: any) {
      setError(err.message || 'Merge failed');
    } finally {
      setMerging(false);
    }
  };

  // 7. Toggle Open/Close PR
  const handleToggleState = async (newState: 'open' | 'closed') => {
    if (!owner || !repoSlug || !numberStr) return;

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/pulls/${numberStr}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ state: newState }),
      });
      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error?.message || 'Failed to update pull request');
      }
      await fetchPR();
    } catch (err: any) {
      setError(err.message);
    }
  };

  if (loading && !pr) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading pull request #{numberStr}...</p>
      </div>
    );
  }

  if (!pr) {
    return (
      <div className="max-w-4xl mx-auto py-16 px-4 text-center">
        <AlertCircle className="w-10 h-10 text-rose-400 mx-auto mb-3" />
        <h2 className="text-lg font-semibold text-white">Pull Request not found</h2>
        <p className="text-xs text-forge-muted mt-1">
          The requested pull request could not be found or you don't have permission to view it.
        </p>
        <Link
          to={`/${owner}/${repoSlug}/pulls`}
          className="btn-primary text-xs inline-flex items-center gap-1.5 mt-4"
        >
          Back to pull requests
        </Link>
      </div>
    );
  }

  const isMerged = pr.state === 'merged';
  const isClosed = pr.state === 'closed';
  const isOpen = pr.state === 'open';

  // Combine comments and reviews into a timeline
  const timelineItems = [
    ...(pr.comments || []).map((c) => ({
      type: 'comment' as const,
      id: c.id,
      author: c.author,
      author_id: c.author_id,
      body: c.body,
      date: new Date(c.created_at).getTime(),
      created_at: c.created_at,
    })),
    ...(pr.reviews || []).map((r) => ({
      type: 'review' as const,
      id: r.id,
      author: r.reviewer,
      author_id: r.reviewer_id,
      state: r.state,
      body: r.body,
      date: new Date(r.created_at).getTime(),
      created_at: r.created_at,
    })),
  ].sort((a, b) => a.date - b.date);

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-16">
      {repo && <RepoHeader repo={repo} activeTab="pulls" />}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        {/* PR Title & Status Header */}
        <div className="space-y-3 pb-4 border-b border-forge-border">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <h1 className="text-xl sm:text-2xl font-bold text-white flex items-center gap-2.5">
              <span>{pr.title}</span>
              <span className="text-forge-muted font-normal text-lg sm:text-xl">
                #{pr.number}
              </span>
            </h1>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setAiReviewOpen(true)}
                className="px-3 py-1.5 rounded-lg bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/30 text-purple-300 hover:text-purple-200 text-xs font-semibold flex items-center gap-1.5 transition-colors shadow-sm"
                title="Run automated security & correctness review with ForgeAI"
              >
                <Sparkles className="w-3.5 h-3.5 text-purple-400" />
                <span>ForgeAI Review</span>
              </button>

              {isOpen && currentUser && (
                <button
                  type="button"
                  onClick={() => setShowReviewBox(!showReviewBox)}
                  className="btn-secondary text-xs flex items-center gap-1.5"
                >
                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                  <span>Review changes</span>
                </button>
              )}
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-xs">
            {/* Status Pill */}
            {isMerged ? (
              <span className="px-3 py-1 rounded-full text-xs font-semibold bg-purple-500/20 text-purple-300 border border-purple-500/30 flex items-center gap-1.5">
                <GitMerge className="w-3.5 h-3.5" /> Merged
              </span>
            ) : isClosed ? (
              <span className="px-3 py-1 rounded-full text-xs font-semibold bg-forge-card text-rose-400 border border-rose-500/30 flex items-center gap-1.5">
                <XCircle className="w-3.5 h-3.5" /> Closed
              </span>
            ) : (
              <span className="px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 flex items-center gap-1.5">
                <GitPullRequest className="w-3.5 h-3.5" /> Open
              </span>
            )}

            <span className="text-forge-muted">
              <strong className="text-white">{pr.author?.username || 'user'}</strong> wants to
              merge {(pr.commits || []).length} commit{pr.commits?.length === 1 ? '' : 's'} into{' '}
              <span className="font-mono bg-forge-card px-2 py-0.5 rounded text-forge-accent border border-forge-border">
                {pr.target_branch}
              </span>{' '}
              from{' '}
              <span className="font-mono bg-forge-card px-2 py-0.5 rounded text-purple-400 border border-forge-border">
                {pr.source_branch}
              </span>
            </span>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
            <AlertCircle className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Tab Navigation */}
        <div className="flex items-center gap-2 border-b border-forge-border pb-px text-xs select-none">
          <button
            type="button"
            onClick={() => setActiveTab('conversation')}
            className={`flex items-center gap-2 px-3 py-2 rounded-t-md font-medium border-b-2 transition ${
              activeTab === 'conversation'
                ? 'border-forge-accent text-white bg-forge-card/40'
                : 'border-transparent text-forge-muted hover:text-white'
            }`}
          >
            <MessageSquare className="w-4 h-4" />
            <span>Conversation</span>
            <span className="px-1.5 py-0.2 rounded-full text-[10px] bg-forge-bg text-forge-muted border border-forge-border">
              {timelineItems.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('commits')}
            className={`flex items-center gap-2 px-3 py-2 rounded-t-md font-medium border-b-2 transition ${
              activeTab === 'commits'
                ? 'border-forge-accent text-white bg-forge-card/40'
                : 'border-transparent text-forge-muted hover:text-white'
            }`}
          >
            <GitCommitIcon className="w-4 h-4" />
            <span>Commits</span>
            <span className="px-1.5 py-0.2 rounded-full text-[10px] bg-forge-bg text-forge-muted border border-forge-border">
              {(pr.commits || []).length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => setActiveTab('files')}
            className={`flex items-center gap-2 px-3 py-2 rounded-t-md font-medium border-b-2 transition ${
              activeTab === 'files'
                ? 'border-forge-accent text-white bg-forge-card/40'
                : 'border-transparent text-forge-muted hover:text-white'
            }`}
          >
            <FileCode className="w-4 h-4" />
            <span>Files changed</span>
            <span className="px-1.5 py-0.2 rounded-full text-[10px] bg-forge-bg text-forge-muted border border-forge-border">
              {pr.diff?.files?.length || 0}
            </span>
          </button>
        </div>

        {/* Tab 1: Conversation & Timeline */}
        {activeTab === 'conversation' && (
          <div className="space-y-6 max-w-4xl">
            {/* PR Description Box */}
            <div className="card overflow-hidden border border-forge-border">
              <div className="px-4 py-2.5 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-white">
                    {pr.author?.username || 'user'}
                  </span>
                  <span className="text-forge-muted">
                    commented on {new Date(pr.created_at).toLocaleString()}
                  </span>
                </div>
                <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-forge-bg text-forge-muted border border-forge-border">
                  Author
                </span>
              </div>
              <div className="p-4 text-sm whitespace-pre-wrap text-white/90">
                {pr.body ? pr.body : <em className="text-forge-muted">No description provided.</em>}
              </div>
            </div>

            {/* Timeline Items (Reviews & Comments) */}
            {timelineItems.map((item) => {
              if (item.type === 'review') {
                const isApproved = item.state === 'APPROVED';
                const isChanges = item.state === 'CHANGES_REQUESTED';

                return (
                  <div key={item.id} className="card overflow-hidden border border-forge-border/80">
                    <div
                      className={`px-4 py-2.5 border-b border-forge-border flex items-center justify-between text-xs ${
                        isApproved
                          ? 'bg-emerald-950/20'
                          : isChanges
                          ? 'bg-rose-950/20'
                          : 'bg-forge-card/80'
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        {isApproved ? (
                          <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                        ) : isChanges ? (
                          <ShieldAlert className="w-4 h-4 text-rose-400 shrink-0" />
                        ) : (
                          <MessageSquare className="w-4 h-4 text-forge-accent shrink-0" />
                        )}
                        <span className="font-semibold text-white">
                          {item.author?.username || 'reviewer'}
                        </span>
                        <span className="text-forge-muted">
                          {isApproved
                            ? 'approved these changes'
                            : isChanges
                            ? 'requested changes'
                            : 'left a review'}{' '}
                          on {new Date(item.created_at).toLocaleString()}
                        </span>
                      </div>
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-semibold border ${
                          isApproved
                            ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30'
                            : isChanges
                            ? 'bg-rose-500/10 text-rose-400 border-rose-500/30'
                            : 'bg-forge-bg text-forge-muted border-forge-border'
                        }`}
                      >
                        {item.state}
                      </span>
                    </div>
                    {item.body && (
                      <div className="p-4 text-sm text-white/90 whitespace-pre-wrap">
                        {item.body}
                      </div>
                    )}
                  </div>
                );
              }

              // Normal PR comment
              return (
                <div key={item.id} className="card overflow-hidden border border-forge-border/80">
                  <div className="px-4 py-2.5 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-white">
                        {item.author?.username || 'user'}
                      </span>
                      <span className="text-forge-muted">
                        commented on {new Date(item.created_at).toLocaleString()}
                      </span>
                    </div>

                    {currentUser &&
                      (currentUser.id === item.author_id || currentUser.is_admin) && (
                        <button
                          type="button"
                          onClick={() => handleDeleteComment(item.id)}
                          className="text-forge-muted hover:text-rose-400 transition"
                          title="Delete comment"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                  </div>
                  <div className="p-4 text-sm text-white/90 whitespace-pre-wrap">
                    {item.body}
                  </div>
                </div>
              );
            })}

            {/* Review Submission Modal / Drawer */}
            {showReviewBox && (
              <div className="card p-4 border border-forge-accent/40 bg-forge-card/90 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-sm font-semibold text-white flex items-center gap-2">
                    <CheckCircle2 className="w-4 h-4 text-forge-accent" />
                    <span>Submit your review</span>
                  </h3>
                  <button
                    type="button"
                    onClick={() => setShowReviewBox(false)}
                    className="text-xs text-forge-muted hover:text-white"
                  >
                    Cancel
                  </button>
                </div>

                <div className="flex flex-wrap gap-2 text-xs">
                  <label
                    className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer select-none transition ${
                      reviewState === 'APPROVED'
                        ? 'bg-emerald-500/20 border-emerald-500/50 text-emerald-300 font-semibold'
                        : 'border-forge-border text-forge-muted hover:text-white'
                    }`}
                  >
                    <input
                      type="radio"
                      name="reviewState"
                      value="APPROVED"
                      checked={reviewState === 'APPROVED'}
                      onChange={() => setReviewState('APPROVED')}
                      className="hidden"
                    />
                    <Check className="w-3.5 h-3.5" />
                    <span>Approve</span>
                  </label>

                  <label
                    className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer select-none transition ${
                      reviewState === 'COMMENTED'
                        ? 'bg-forge-accent/20 border-forge-accent/50 text-white font-semibold'
                        : 'border-forge-border text-forge-muted hover:text-white'
                    }`}
                  >
                    <input
                      type="radio"
                      name="reviewState"
                      value="COMMENTED"
                      checked={reviewState === 'COMMENTED'}
                      onChange={() => setReviewState('COMMENTED')}
                      className="hidden"
                    />
                    <MessageSquare className="w-3.5 h-3.5" />
                    <span>Comment</span>
                  </label>

                  <label
                    className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer select-none transition ${
                      reviewState === 'CHANGES_REQUESTED'
                        ? 'bg-rose-500/20 border-rose-500/50 text-rose-300 font-semibold'
                        : 'border-forge-border text-forge-muted hover:text-white'
                    }`}
                  >
                    <input
                      type="radio"
                      name="reviewState"
                      value="CHANGES_REQUESTED"
                      checked={reviewState === 'CHANGES_REQUESTED'}
                      onChange={() => setReviewState('CHANGES_REQUESTED')}
                      className="hidden"
                    />
                    <ShieldAlert className="w-3.5 h-3.5" />
                    <span>Request changes</span>
                  </label>
                </div>

                <textarea
                  rows={3}
                  placeholder="Leave a review comment..."
                  value={reviewBody}
                  onChange={(e) => setReviewBody(e.target.value)}
                  className="w-full px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-xs text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent font-mono"
                />

                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={handleSubmitReview}
                    disabled={submittingReview}
                    className="btn-primary text-xs flex items-center gap-1.5"
                  >
                    {submittingReview && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                    <span>Submit review</span>
                  </button>
                </div>
              </div>
            )}

            {/* Merge Status & Action Box */}
            <div className="card p-5 border border-forge-border space-y-4">
              {isMerged ? (
                <div className="flex items-center gap-3 text-purple-300 text-sm font-medium">
                  <GitMerge className="w-5 h-5 text-purple-400 shrink-0" />
                  <div>
                    <p>Pull request successfully merged and closed.</p>
                    {pr.merge_commit_sha && (
                      <p className="text-xs text-forge-muted font-mono mt-0.5">
                        Merge commit: {pr.merge_commit_sha.slice(0, 7)}
                      </p>
                    )}
                  </div>
                </div>
              ) : isClosed ? (
                <div className="flex items-center justify-between gap-4">
                  <div className="flex items-center gap-2.5 text-rose-400 text-sm">
                    <XCircle className="w-5 h-5 shrink-0" />
                    <span>This pull request is closed without merging.</span>
                  </div>
                  {currentUser && (
                    <button
                      type="button"
                      onClick={() => handleToggleState('open')}
                      className="btn-secondary text-xs"
                    >
                      Reopen pull request
                    </button>
                  )}
                </div>
              ) : (
                <div className="space-y-4">
                  {/* Mergeability check banner */}
                  {pr.can_merge ? (
                    <div className="flex items-center gap-3 text-emerald-300 text-xs font-medium">
                      <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
                      <div>
                        <p className="font-semibold text-white">This branch has no conflicts with the base branch</p>
                        <p className="text-forge-muted mt-0.5">Merging can be performed automatically.</p>
                      </div>
                    </div>
                  ) : (
                    <div className="flex items-center gap-3 text-amber-300 text-xs font-medium">
                      <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
                      <div>
                        <p className="font-semibold text-white">This branch has conflicts that must be resolved</p>
                        <p className="text-forge-muted mt-0.5">
                          Conflicting files must be reconciled before merging can proceed.
                        </p>
                      </div>
                    </div>
                  )}

                  {/* Merge Method Selector & Commit Message Form */}
                  {pr.can_merge && (
                    <div className="space-y-3 pt-2 border-t border-forge-border/60">
                      <div className="flex flex-wrap gap-2 text-xs">
                        <button
                          type="button"
                          onClick={() => setMergeMethod('merge')}
                          className={`px-3 py-1.5 rounded-lg border font-medium transition ${
                            mergeMethod === 'merge'
                              ? 'bg-forge-accent/20 border-forge-accent text-white'
                              : 'border-forge-border text-forge-muted hover:text-white'
                          }`}
                        >
                          Create a merge commit
                        </button>

                        <button
                          type="button"
                          onClick={() => setMergeMethod('squash')}
                          className={`px-3 py-1.5 rounded-lg border font-medium transition ${
                            mergeMethod === 'squash'
                              ? 'bg-forge-accent/20 border-forge-accent text-white'
                              : 'border-forge-border text-forge-muted hover:text-white'
                          }`}
                        >
                          Squash and merge
                        </button>
                      </div>

                      <input
                        type="text"
                        value={commitMessage}
                        onChange={(e) => setCommitMessage(e.target.value)}
                        placeholder="Commit message..."
                        className="w-full px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-xs text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent font-mono"
                      />

                      <div className="flex items-center justify-between pt-1">
                        <button
                          type="button"
                          onClick={handleMerge}
                          disabled={merging}
                          className="btn-primary text-xs flex items-center gap-2 bg-purple-600 hover:bg-purple-500 border-purple-500/50"
                        >
                          {merging ? (
                            <Loader2 className="w-3.5 h-3.5 animate-spin" />
                          ) : (
                            <GitMerge className="w-3.5 h-3.5" />
                          )}
                          <span>
                            {mergeMethod === 'squash' ? 'Confirm squash and merge' : 'Merge pull request'}
                          </span>
                        </button>

                        {currentUser && (
                          <button
                            type="button"
                            onClick={() => handleToggleState('closed')}
                            className="btn-secondary text-xs text-rose-400 hover:text-rose-300"
                          >
                            Close pull request
                          </button>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Quick Comment Input */}
            {currentUser && (
              <form onSubmit={handleAddComment} className="card p-4 border border-forge-border space-y-3">
                <h3 className="text-xs font-semibold text-white">Add a comment</h3>
                <textarea
                  rows={3}
                  required
                  placeholder="Leave a comment..."
                  value={commentBody}
                  onChange={(e) => setCommentBody(e.target.value)}
                  className="w-full px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-xs text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent font-mono"
                />
                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={submittingComment || !commentBody.trim()}
                    className="btn-primary text-xs flex items-center gap-1.5 disabled:opacity-50"
                  >
                    {submittingComment ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Send className="w-3.5 h-3.5" />
                    )}
                    <span>Comment</span>
                  </button>
                </div>
              </form>
            )}
          </div>
        )}

        {/* Tab 2: Commits */}
        {activeTab === 'commits' && (
          <div className="card overflow-hidden border border-forge-border max-w-4xl">
            <div className="px-4 py-3 bg-forge-card/80 border-b border-forge-border flex items-center gap-2 text-xs font-semibold text-white">
              <GitCommitIcon className="w-4 h-4 text-forge-accent" />
              <span>Commits ({(pr.commits || []).length})</span>
            </div>
            {pr.commits && pr.commits.length > 0 ? (
              <div className="divide-y divide-forge-border/40 text-xs">
                {pr.commits.map((c: GitCommit) => (
                  <div key={c.sha} className="p-4 hover:bg-forge-card/40 flex items-center justify-between gap-4">
                    <div className="space-y-1 min-w-0">
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
            ) : (
              <div className="p-8 text-center text-xs text-forge-muted">
                No commits found between branches.
              </div>
            )}
          </div>
        )}

        {/* Tab 3: Files Changed & Diff Viewer */}
        {activeTab === 'files' && (
          <div>
            {pr.diff ? (
              <DiffViewer diff={pr.diff} />
            ) : (
              <div className="card p-8 text-center text-forge-muted text-xs">
                Loading diff changes...
              </div>
            )}
          </div>
        )}
      </div>

      {owner && repoSlug && pr && (
        <AIReviewModal
          isOpen={aiReviewOpen}
          onClose={() => setAiReviewOpen(false)}
          owner={owner}
          repo={repoSlug}
          pullNumber={pr.number}
          onReviewPosted={() => {
            fetchPR();
          }}
        />
      )}
    </div>
  );
};
