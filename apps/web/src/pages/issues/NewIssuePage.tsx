import React, { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { Repository, Label, Milestone } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import { useAuth } from '../../context/AuthContext';
import {
  CircleDot,
  Loader2,
  AlertCircle,
  Tag,
  Milestone as MilestoneIcon,
  Check,
  Eye,
  Edit3
} from 'lucide-react';

export const NewIssuePage: React.FC = () => {
  const { owner, repo: repoSlug } = useParams<{ owner: string; repo: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [labels, setLabels] = useState<Label[]>([]);
  const [milestones, setMilestones] = useState<Milestone[]>([]);

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [selectedLabels, setSelectedLabels] = useState<string[]>([]);
  const [selectedMilestone, setSelectedMilestone] = useState<string>('');

  const [activeTab, setActiveTab] = useState<'write' | 'preview'>('write');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

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

    fetch(`/api/v1/repos/${owner}/${repoSlug}/labels`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => setLabels(data?.labels || []))
      .catch(() => {});

    fetch(`/api/v1/repos/${owner}/${repoSlug}/milestones`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => setMilestones(data?.milestones || []))
      .catch(() => {});
  }, [owner, repoSlug]);

  const toggleLabel = (labelId: string) => {
    setSelectedLabels((prev) =>
      prev.includes(labelId) ? prev.filter((id) => id !== labelId) : [...prev, labelId]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError('Please provide an issue title.');
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/issues`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          title: title.trim(),
          body: body.trim(),
          label_ids: selectedLabels,
          milestone_id: selectedMilestone ? selectedMilestone : undefined,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to create issue.');
      }

      navigate(`/${owner}/${repoSlug}/issues/${data.issue.number}`);
    } catch (err: any) {
      setError(err.message || 'An error occurred.');
    } finally {
      setSubmitting(false);
    }
  };

  if (!repo) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading repository...</p>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <CircleDot className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Authentication Required</h2>
        <p className="text-forge-muted mb-6">You must be logged in to open a new issue.</p>
        <Link to="/login" className="btn-primary">Sign In</Link>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-forge-bg pb-16">
      <RepoHeader repo={repo} activeTab="issues" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-6">
          <h1 className="text-xl font-bold text-white flex items-center gap-2">
            <CircleDot className="w-5 h-5 text-emerald-400" />
            Create a New Issue
          </h1>
          <p className="text-xs text-forge-muted mt-1">
            Track a bug, propose an enhancement, or open a task for discussion.
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-950/40 border border-red-800/50 rounded-xl text-red-200 text-sm flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="grid grid-cols-1 lg:grid-cols-4 gap-8">
          {/* Main Column: Title & Body */}
          <div className="lg:col-span-3 space-y-4">
            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
                Issue Title <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                required
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Title summarizing the issue"
                className="input-field text-sm font-medium py-2.5"
              />
            </div>

            {/* Description Tabbed Box */}
            <div className="card overflow-hidden">
              <div className="flex items-center justify-between px-3 py-2 border-b border-forge-border bg-forge-card/60">
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={() => setActiveTab('write')}
                    className={`flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                      activeTab === 'write'
                        ? 'bg-forge-surface text-white'
                        : 'text-forge-muted hover:text-white'
                    }`}
                  >
                    <Edit3 className="w-3.5 h-3.5" />
                    <span>Write</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setActiveTab('preview')}
                    className={`flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                      activeTab === 'preview'
                        ? 'bg-forge-surface text-white'
                        : 'text-forge-muted hover:text-white'
                    }`}
                  >
                    <Eye className="w-3.5 h-3.5" />
                    <span>Preview</span>
                  </button>
                </div>

                <span className="text-[11px] text-forge-muted select-none">Markdown supported</span>
              </div>

              {activeTab === 'write' ? (
                <textarea
                  rows={10}
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  placeholder="Leave a comment or describe the steps to reproduce..."
                  className="w-full bg-transparent p-4 text-xs text-forge-text placeholder:text-forge-muted/60 focus:outline-none resize-y font-mono leading-relaxed"
                />
              ) : (
                <div className="p-6 min-h-[220px] text-sm text-forge-text leading-relaxed whitespace-pre-wrap font-sans">
                  {body.trim() ? body : <span className="text-forge-muted italic">Nothing to preview</span>}
                </div>
              )}
            </div>

            <div className="flex items-center justify-between pt-2">
              <Link to={`/${owner}/${repoSlug}/issues`} className="btn-secondary text-xs">
                Cancel
              </Link>
              <button
                type="submit"
                disabled={submitting}
                className="btn-primary text-xs flex items-center gap-2"
              >
                {submitting ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Submitting Issue...</span>
                  </>
                ) : (
                  <span>Submit New Issue</span>
                )}
              </button>
            </div>
          </div>

          {/* Right Sidebar: Labels & Milestone */}
          <div className="lg:col-span-1 space-y-6">
            {/* Labels widget */}
            <div className="card p-4 space-y-3">
              <div className="flex items-center gap-2 text-xs font-semibold text-white uppercase tracking-wider">
                <Tag className="w-4 h-4 text-forge-muted" />
                <span>Labels</span>
              </div>

              {labels.length === 0 ? (
                <p className="text-xs text-forge-muted">No labels created yet.</p>
              ) : (
                <div className="space-y-1.5 max-h-60 overflow-y-auto pr-1">
                  {labels.map((l) => {
                    const isSelected = selectedLabels.includes(l.id);
                    return (
                      <button
                        key={l.id}
                        type="button"
                        onClick={() => toggleLabel(l.id)}
                        className={`w-full flex items-center justify-between p-2 rounded-lg text-left transition-colors text-xs ${
                          isSelected ? 'bg-forge-surface border border-forge-border' : 'hover:bg-forge-card/50'
                        }`}
                      >
                        <span
                          style={{
                            backgroundColor: `${l.color}25`,
                            borderColor: `${l.color}60`,
                            color: l.color,
                          }}
                          className="px-2 py-0.5 rounded-full text-[10px] font-medium border"
                        >
                          {l.name}
                        </span>
                        {isSelected && <Check className="w-3.5 h-3.5 text-blue-400" />}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>

            {/* Milestone widget */}
            <div className="card p-4 space-y-3">
              <div className="flex items-center gap-2 text-xs font-semibold text-white uppercase tracking-wider">
                <MilestoneIcon className="w-4 h-4 text-forge-muted" />
                <span>Milestone</span>
              </div>

              {milestones.length === 0 ? (
                <p className="text-xs text-forge-muted">No milestones defined.</p>
              ) : (
                <select
                  value={selectedMilestone}
                  onChange={(e) => setSelectedMilestone(e.target.value)}
                  className="input-field text-xs cursor-pointer"
                >
                  <option value="">No milestone</option>
                  {milestones.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.title}
                    </option>
                  ))}
                </select>
              )}
            </div>
          </div>
        </form>
      </div>
    </div>
  );
};
