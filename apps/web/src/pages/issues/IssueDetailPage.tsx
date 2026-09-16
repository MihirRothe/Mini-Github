import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, IssueDetail } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import { useAuth } from '../../context/AuthContext';
import {
  CircleDot,
  CheckCircle2,
  Tag,
  Milestone as MilestoneIcon,
  Edit3,
  Eye,
  Trash2,
  Loader2,
  Send,
  Plus
} from 'lucide-react';

export const IssueDetailPage: React.FC = () => {
  const { owner, repo: repoSlug, number: numberStr } = useParams<{
    owner: string;
    repo: string;
    number: string;
  }>();
  const { user } = useAuth();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [issue, setIssue] = useState<IssueDetail | null>(null);

  const [newCommentBody, setNewCommentBody] = useState('');
  const [activeTab, setActiveTab] = useState<'write' | 'preview'>('write');
  const [commenting, setCommenting] = useState(false);
  const [togglingState, setTogglingState] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchIssueDetail = async () => {
    if (!owner || !repoSlug || !numberStr) return;
    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/issues/${numberStr}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load issue.');
      }
      setIssue(data.issue);
    } catch (err: any) {
      setError(err.message || 'Error fetching issue.');
    }
  };

  useEffect(() => {
    if (!owner || !repoSlug) return;
    fetch(`/api/v1/repos/${owner}/${repoSlug}`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.repository) setRepo(data.repository);
      })
      .catch(() => {});

    fetchIssueDetail();
  }, [owner, repoSlug, numberStr]);

  const handleAddComment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCommentBody.trim()) return;

    setCommenting(true);
    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/issues/${numberStr}/comments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ body: newCommentBody.trim() }),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error?.message || 'Failed to post comment.');
      }
      setNewCommentBody('');
      setActiveTab('write');
      await fetchIssueDetail();
    } catch (err: any) {
      alert(err.message || 'Could not post comment.');
    } finally {
      setCommenting(false);
    }
  };

  const handleToggleState = async () => {
    if (!issue) return;
    const newState = issue.state === 'open' ? 'closed' : 'open';

    setTogglingState(true);
    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/issues/${numberStr}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ state: newState }),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error?.message || 'Failed to update issue state.');
      }
      await fetchIssueDetail();
    } catch (err: any) {
      alert(err.message || 'Error updating state.');
    } finally {
      setTogglingState(false);
    }
  };

  const handleDeleteComment = async (commentId: string) => {
    if (!confirm('Are you sure you want to delete this comment?')) return;
    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repoSlug}/issues/${numberStr}/comments/${commentId}`,
        {
          method: 'DELETE',
          credentials: 'include',
        }
      );
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error?.message || 'Failed to delete comment.');
      }
      await fetchIssueDetail();
    } catch (err: any) {
      alert(err.message || 'Error deleting comment.');
    }
  };

  if (!repo || !issue) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        {error ? (
          <div className="text-rose-400 text-sm max-w-md mx-auto">{error}</div>
        ) : (
          <>
            <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
            <p className="text-forge-muted text-sm">Loading issue #{numberStr}...</p>
          </>
        )}
      </div>
    );
  }

  const isAuthor = user && issue.author && user.username === issue.author.username;
  const canModifyState = isAuthor || (repo.current_user_permission && ['triage', 'write', 'maintain', 'admin'].includes(repo.current_user_permission));

  return (
    <div className="min-h-screen bg-forge-bg pb-16">
      <RepoHeader repo={repo} activeTab="issues" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        {/* Issue Title Header */}
        <div className="border-b border-forge-border pb-6 space-y-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <h1 className="text-2xl font-bold text-white flex items-center gap-3">
              <span>{issue.title}</span>
              <span className="text-forge-muted font-normal">#{issue.number}</span>
            </h1>

            <Link to={`/${owner}/${repoSlug}/issues/new`} className="btn-primary text-xs self-start sm:self-auto flex items-center gap-1.5">
              <Plus className="w-3.5 h-3.5" />
              <span>New Issue</span>
            </Link>
          </div>

          <div className="flex flex-wrap items-center gap-3 text-xs text-forge-muted">
            {/* State badge */}
            <span
              className={`flex items-center gap-1.5 px-2.5 py-1 rounded-full font-semibold text-xs text-white ${
                issue.state === 'open' ? 'bg-emerald-600' : 'bg-purple-600'
              }`}
            >
              {issue.state === 'open' ? (
                <>
                  <CircleDot className="w-3.5 h-3.5" />
                  Open
                </>
              ) : (
                <>
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  Closed
                </>
              )}
            </span>

            <span>
              {issue.author?.username || 'someone'} opened this issue on{' '}
              {new Date(issue.created_at).toLocaleDateString()}
            </span>
            <span>·</span>
            <span>{issue.comments.length} comments</span>

            {issue.state === 'closed' && issue.closed_at && (
              <>
                <span>·</span>
                <span>
                  Closed on {new Date(issue.closed_at).toLocaleDateString()}
                  {issue.closed_by && ` by ${issue.closed_by.username}`}
                </span>
              </>
            )}
          </div>
        </div>

        {/* Main Grid: Discussion Thread + Sidebar */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          {/* Left: Thread */}
          <div className="lg:col-span-3 space-y-6">
            {/* Original Issue Description Card */}
            <div className="card overflow-hidden">
              <div className="p-3 px-4 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs">
                <div className="flex items-center gap-2.5">
                  <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-blue-600 to-indigo-600 flex items-center justify-center text-[10px] font-bold text-white uppercase select-none">
                    {(issue.author?.username || 'A').slice(0, 1)}
                  </div>
                  <span className="font-semibold text-white">{issue.author?.username || 'Anonymous'}</span>
                  <span className="text-forge-muted">
                    commented on {new Date(issue.created_at).toLocaleDateString()}
                  </span>
                </div>
                <span className="px-2 py-0.5 rounded text-[10px] border border-forge-border bg-forge-bg text-forge-muted">
                  Author
                </span>
              </div>

              <div className="p-6 text-sm text-forge-text leading-relaxed whitespace-pre-wrap font-sans">
                {issue.body || <span className="italic text-forge-muted">No description provided.</span>}
              </div>
            </div>

            {/* Comments Timeline */}
            {issue.comments.map((comment) => {
              const isCommentAuthor = user && comment.author && user.username === comment.author.username;
              return (
                <div key={comment.id} className="card overflow-hidden">
                  <div className="p-3 px-4 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs">
                    <div className="flex items-center gap-2.5">
                      <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-purple-600 to-pink-600 flex items-center justify-center text-[10px] font-bold text-white uppercase select-none">
                        {(comment.author?.username || 'U').slice(0, 1)}
                      </div>
                      <span className="font-semibold text-white">{comment.author?.username || 'Anonymous'}</span>
                      <span className="text-forge-muted">
                        commented on {new Date(comment.created_at).toLocaleDateString()}
                      </span>
                    </div>

                    {isCommentAuthor && (
                      <button
                        onClick={() => handleDeleteComment(comment.id)}
                        className="text-forge-muted hover:text-rose-400 transition-colors p-1"
                        title="Delete comment"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>

                  <div className="p-6 text-sm text-forge-text leading-relaxed whitespace-pre-wrap font-sans">
                    {comment.body}
                  </div>
                </div>
              );
            })}

            {/* Add Comment Box */}
            {user ? (
              <form onSubmit={handleAddComment} className="card overflow-hidden">
                <div className="flex items-center justify-between px-3 py-2 border-b border-forge-border bg-forge-card/60">
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setActiveTab('write')}
                      className={`flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                        activeTab === 'write' ? 'bg-forge-surface text-white' : 'text-forge-muted hover:text-white'
                      }`}
                    >
                      <Edit3 className="w-3.5 h-3.5" />
                      <span>Write</span>
                    </button>
                    <button
                      type="button"
                      onClick={() => setActiveTab('preview')}
                      className={`flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                        activeTab === 'preview' ? 'bg-forge-surface text-white' : 'text-forge-muted hover:text-white'
                      }`}
                    >
                      <Eye className="w-3.5 h-3.5" />
                      <span>Preview</span>
                    </button>
                  </div>
                  <span className="text-[11px] text-forge-muted">Markdown supported</span>
                </div>

                {activeTab === 'write' ? (
                  <textarea
                    rows={4}
                    value={newCommentBody}
                    onChange={(e) => setNewCommentBody(e.target.value)}
                    placeholder="Leave a comment..."
                    className="w-full bg-transparent p-4 text-xs text-forge-text placeholder:text-forge-muted/60 focus:outline-none resize-y font-mono leading-relaxed"
                  />
                ) : (
                  <div className="p-6 min-h-[100px] text-sm text-forge-text leading-relaxed whitespace-pre-wrap font-sans">
                    {newCommentBody.trim() ? newCommentBody : <span className="text-forge-muted italic">Nothing to preview</span>}
                  </div>
                )}

                <div className="p-3 bg-forge-card/40 border-t border-forge-border flex items-center justify-between">
                  {canModifyState ? (
                    <button
                      type="button"
                      disabled={togglingState}
                      onClick={handleToggleState}
                      className={`btn-secondary text-xs flex items-center gap-1.5 ${
                        issue.state === 'open'
                          ? 'hover:border-purple-500 hover:text-purple-300'
                          : 'hover:border-emerald-500 hover:text-emerald-300'
                      }`}
                    >
                      {issue.state === 'open' ? (
                        <>
                          <CheckCircle2 className="w-3.5 h-3.5 text-purple-400" />
                          <span>Close Issue</span>
                        </>
                      ) : (
                        <>
                          <CircleDot className="w-3.5 h-3.5 text-emerald-400" />
                          <span>Reopen Issue</span>
                        </>
                      )}
                    </button>
                  ) : <div />}

                  <button
                    type="submit"
                    disabled={commenting || !newCommentBody.trim()}
                    className="btn-primary text-xs flex items-center gap-1.5"
                  >
                    {commenting ? (
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    ) : (
                      <Send className="w-3.5 h-3.5" />
                    )}
                    <span>Comment</span>
                  </button>
                </div>
              </form>
            ) : (
              <div className="card p-6 text-center space-y-2">
                <p className="text-xs text-forge-muted">
                  Sign in to join this conversation and leave a comment.
                </p>
                <Link to="/login" className="btn-secondary text-xs inline-block">
                  Sign In
                </Link>
              </div>
            )}
          </div>

          {/* Right Sidebar */}
          <div className="lg:col-span-1 space-y-6">
            {/* Labels widget */}
            <div className="card p-4 space-y-3">
              <div className="flex items-center gap-2 text-xs font-semibold text-white uppercase tracking-wider">
                <Tag className="w-4 h-4 text-forge-muted" />
                <span>Labels</span>
              </div>

              {issue.labels.length === 0 ? (
                <p className="text-xs text-forge-muted">None yet</p>
              ) : (
                <div className="flex flex-wrap gap-1.5">
                  {issue.labels.map((l) => (
                    <span
                      key={l.id}
                      style={{
                        backgroundColor: `${l.color}25`,
                        borderColor: `${l.color}60`,
                        color: l.color,
                      }}
                      className="px-2 py-0.5 rounded-full text-[10px] font-medium border"
                    >
                      {l.name}
                    </span>
                  ))}
                </div>
              )}
            </div>

            {/* Milestone widget */}
            <div className="card p-4 space-y-3">
              <div className="flex items-center gap-2 text-xs font-semibold text-white uppercase tracking-wider">
                <MilestoneIcon className="w-4 h-4 text-forge-muted" />
                <span>Milestone</span>
              </div>

              {issue.milestone ? (
                <div className="space-y-1">
                  <div className="text-xs font-medium text-white">{issue.milestone.title}</div>
                  {issue.milestone.due_date && (
                    <p className="text-[11px] text-forge-muted">
                      Due by {new Date(issue.milestone.due_date).toLocaleDateString()}
                    </p>
                  )}
                </div>
              ) : (
                <p className="text-xs text-forge-muted">No milestone</p>
              )}
            </div>

            {/* Assignees widget */}
            <div className="card p-4 space-y-3">
              <div className="text-xs font-semibold text-white uppercase tracking-wider">
                Assignees
              </div>
              {issue.assignees.length === 0 ? (
                <p className="text-xs text-forge-muted">No one assigned</p>
              ) : (
                <div className="space-y-2">
                  {issue.assignees.map((a) => (
                    <div key={a.id} className="flex items-center gap-2 text-xs text-white">
                      <div className="w-5 h-5 rounded-full bg-gradient-to-tr from-blue-600 to-indigo-600 flex items-center justify-center text-[9px] font-bold text-white uppercase">
                        {a.username.slice(0, 1)}
                      </div>
                      <span>{a.username}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
