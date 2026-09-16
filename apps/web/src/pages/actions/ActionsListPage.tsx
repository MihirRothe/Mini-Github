import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, PipelineRun } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import {
  PlayCircle,
  CheckCircle2,
  XCircle,
  Loader2,
  Clock,
  Ban,
  GitBranch,
  GitCommit as GitCommitIcon,
  Play,
  Filter,
  AlertCircle
} from 'lucide-react';

export const ActionsListPage: React.FC = () => {
  const { owner, repo: repoSlug } = useParams<{ owner: string; repo: string }>();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [runs, setRuns] = useState<PipelineRun[]>([]);
  const [total, setTotal] = useState(0);
  const [workflows, setWorkflows] = useState<string[]>([]);
  const [branches, setBranches] = useState<string[]>([]);

  // Filter state
  const [selectedWorkflow, setSelectedWorkflow] = useState<string>('all');
  const [statusFilter, setStatusFilter] = useState<string>('all');

  // Trigger modal state
  const [showTriggerModal, setShowTriggerModal] = useState(false);
  const [selectedBranch, setSelectedBranch] = useState<string>('main');
  const [selectedWorkflowFile, setSelectedWorkflowFile] = useState<string>('forge-ci.yml');
  const [triggering, setTriggering] = useState(false);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 1. Load Repo, Workflows & Branches
  useEffect(() => {
    if (!owner || !repoSlug) return;

    fetch(`/api/v1/repos/${owner}/${repoSlug}`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.repository) {
          setRepo(data.repository);
          setSelectedBranch(data.repository.default_branch || 'main');
        }
      })
      .catch(() => {});

    fetch(`/api/v1/repos/${owner}/${repoSlug}/actions/workflows`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.workflows) {
          setWorkflows(data.workflows);
          if (data.workflows.length > 0) {
            setSelectedWorkflowFile(data.workflows[0]);
          }
        }
      })
      .catch(() => {});

    fetch(`/api/v1/repos/${owner}/${repoSlug}/branches`, { credentials: 'include' })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.branches) {
          setBranches(data.branches.map((b: any) => b.name));
        }
      })
      .catch(() => {});
  }, [owner, repoSlug]);

  // 2. Load Pipeline Runs
  const fetchRuns = async () => {
    if (!owner || !repoSlug) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    if (statusFilter !== 'all') params.set('status', statusFilter);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/actions/runs?${params.toString()}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load pipeline runs');
      }
      setRuns(data.runs || []);
      setTotal(data.total || 0);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch runs');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRuns();
  }, [owner, repoSlug, statusFilter]);

  // 3. Trigger manual workflow
  const handleTrigger = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!owner || !repoSlug) return;

    setTriggering(true);
    setError(null);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/actions/runs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          branch: selectedBranch,
          workflow_file: selectedWorkflowFile,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to trigger workflow');
      }
      setShowTriggerModal(false);
      await fetchRuns();
    } catch (err: any) {
      setError(err.message || 'Trigger failed');
    } finally {
      setTriggering(false);
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />;
      case 'failure':
        return <XCircle className="w-5 h-5 text-rose-400 shrink-0" />;
      case 'running':
        return <Loader2 className="w-5 h-5 text-amber-400 animate-spin shrink-0" />;
      case 'queued':
        return <Clock className="w-5 h-5 text-forge-muted shrink-0" />;
      case 'cancelled':
        return <Ban className="w-5 h-5 text-forge-muted shrink-0" />;
      default:
        return <PlayCircle className="w-5 h-5 text-forge-accent shrink-0" />;
    }
  };

  const formatDuration = (ms: number, status: string) => {
    if (status === 'running') return 'running...';
    if (status === 'queued') return 'queued';
    if (!ms || ms <= 0) return '0s';
    const totalSecs = Math.round(ms / 1000);
    if (totalSecs < 60) return `${totalSecs}s`;
    const mins = Math.floor(totalSecs / 60);
    const remSecs = totalSecs % 60;
    return `${mins}m ${remSecs}s`;
  };

  const filteredRuns = runs.filter((r) => {
    if (selectedWorkflow !== 'all' && r.workflow_file !== selectedWorkflow && r.workflow_name !== selectedWorkflow) {
      return false;
    }
    return true;
  });

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-16">
      {repo && <RepoHeader repo={repo} activeTab="actions" />}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        {/* Top Header & Trigger Action */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-bold text-white flex items-center gap-2">
              <PlayCircle className="w-5 h-5 text-forge-accent" />
              <span>Actions & CI/CD Pipelines</span>
            </h1>
            <p className="text-xs text-forge-muted mt-1">
              Automated builds, container execution, testing, and continuous deployment workflows.
            </p>
          </div>

          <div className="relative">
            <button
              type="button"
              onClick={() => setShowTriggerModal(!showTriggerModal)}
              className="btn-primary text-xs flex items-center gap-1.5 shadow-lg shadow-forge-accent/10"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Run workflow</span>
            </button>

            {/* Run Workflow Modal / Dropdown */}
            {showTriggerModal && (
              <div className="absolute right-0 mt-2 w-80 bg-forge-card border border-forge-border rounded-xl shadow-2xl p-4 z-50 space-y-4">
                <div className="flex items-center justify-between border-b border-forge-border pb-2">
                  <h3 className="text-xs font-semibold text-white">Trigger Workflow</h3>
                  <button
                    type="button"
                    onClick={() => setShowTriggerModal(false)}
                    className="text-xs text-forge-muted hover:text-white"
                  >
                    ✕
                  </button>
                </div>

                <form onSubmit={handleTrigger} className="space-y-3">
                  <div>
                    <label className="block text-[11px] font-medium text-forge-muted mb-1">
                      Branch
                    </label>
                    <select
                      value={selectedBranch}
                      onChange={(e) => setSelectedBranch(e.target.value)}
                      className="w-full px-2.5 py-1.5 bg-forge-bg border border-forge-border rounded-lg text-xs text-white focus:outline-none focus:border-forge-accent font-mono"
                    >
                      {branches.length > 0 ? (
                        branches.map((b) => (
                          <option key={b} value={b}>
                            {b}
                          </option>
                        ))
                      ) : (
                        <option value="main">main</option>
                      )}
                    </select>
                  </div>

                  <div>
                    <label className="block text-[11px] font-medium text-forge-muted mb-1">
                      Workflow File
                    </label>
                    <select
                      value={selectedWorkflowFile}
                      onChange={(e) => setSelectedWorkflowFile(e.target.value)}
                      className="w-full px-2.5 py-1.5 bg-forge-bg border border-forge-border rounded-lg text-xs text-white focus:outline-none focus:border-forge-accent font-mono"
                    >
                      {workflows.length > 0 ? (
                        workflows.map((w) => (
                          <option key={w} value={w}>
                            {w}
                          </option>
                        ))
                      ) : (
                        <option value="forge-ci.yml">forge-ci.yml (Default CI)</option>
                      )}
                    </select>
                  </div>

                  <div className="pt-2 flex justify-end">
                    <button
                      type="submit"
                      disabled={triggering}
                      className="btn-primary text-xs flex items-center gap-1.5 w-full justify-center"
                    >
                      {triggering ? (
                        <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      ) : (
                        <Play className="w-3.5 h-3.5 fill-current" />
                      )}
                      <span>Run workflow</span>
                    </button>
                  </div>
                </form>
              </div>
            )}
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
            <AlertCircle className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Main Grid: Sidebar Filters + Runs List */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar */}
          <div className="space-y-4">
            <div className="card p-4 space-y-3 border border-forge-border">
              <h2 className="text-xs font-semibold text-white uppercase tracking-wider flex items-center gap-2">
                <Filter className="w-3.5 h-3.5 text-forge-accent" />
                <span>Workflows</span>
              </h2>

              <div className="space-y-1 text-xs">
                <button
                  type="button"
                  onClick={() => setSelectedWorkflow('all')}
                  className={`w-full text-left px-2.5 py-1.5 rounded-lg transition ${
                    selectedWorkflow === 'all'
                      ? 'bg-forge-accent/20 text-white font-medium'
                      : 'text-forge-muted hover:text-white hover:bg-forge-card'
                  }`}
                >
                  All workflows
                </button>

                {workflows.map((wf) => (
                  <button
                    key={wf}
                    type="button"
                    onClick={() => setSelectedWorkflow(wf)}
                    className={`w-full text-left px-2.5 py-1.5 rounded-lg truncate transition font-mono ${
                      selectedWorkflow === wf
                        ? 'bg-forge-accent/20 text-white font-medium'
                        : 'text-forge-muted hover:text-white hover:bg-forge-card'
                    }`}
                  >
                    {wf}
                  </button>
                ))}
              </div>
            </div>

            {/* Status Filter */}
            <div className="card p-4 space-y-3 border border-forge-border">
              <h2 className="text-xs font-semibold text-white uppercase tracking-wider">Status</h2>
              <div className="space-y-1 text-xs">
                {['all', 'success', 'failure', 'running', 'queued', 'cancelled'].map((st) => (
                  <button
                    key={st}
                    type="button"
                    onClick={() => setStatusFilter(st)}
                    className={`w-full text-left px-2.5 py-1.5 rounded-lg capitalize transition ${
                      statusFilter === st
                        ? 'bg-forge-accent/20 text-white font-medium'
                        : 'text-forge-muted hover:text-white hover:bg-forge-card'
                    }`}
                  >
                    {st}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* Runs Content */}
          <div className="lg:col-span-3 space-y-4">
            <div className="card overflow-hidden border border-forge-border">
              <div className="px-4 py-3 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs font-medium text-white">
                <span>Pipeline Runs ({total})</span>
                <button
                  type="button"
                  onClick={fetchRuns}
                  className="text-forge-muted hover:text-white transition"
                >
                  Refresh
                </button>
              </div>

              {loading && runs.length === 0 ? (
                <div className="p-12 text-center text-forge-muted">
                  <Loader2 className="w-6 h-6 text-forge-accent animate-spin mx-auto mb-2" />
                  <p className="text-xs">Fetching pipeline runs...</p>
                </div>
              ) : filteredRuns.length === 0 ? (
                <div className="p-12 text-center text-forge-muted space-y-3">
                  <PlayCircle className="w-10 h-10 mx-auto text-forge-muted/40" />
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-white">No pipeline runs yet</p>
                    <p className="text-xs text-forge-muted max-w-sm mx-auto">
                      ForgeHub CI executes automated build, test, and container pipelines on pushes or manual triggers.
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setShowTriggerModal(true)}
                    className="btn-primary text-xs inline-flex items-center gap-1.5 mt-2"
                  >
                    <Play className="w-3.5 h-3.5 fill-current" />
                    Trigger First Workflow
                  </button>
                </div>
              ) : (
                <div className="divide-y divide-forge-border/60">
                  {filteredRuns.map((run) => (
                    <div
                      key={run.id}
                      className="p-4 hover:bg-forge-card/40 transition flex items-start justify-between gap-4"
                    >
                      <div className="flex items-start gap-3.5 min-w-0">
                        <div className="mt-0.5">{getStatusIcon(run.status)}</div>

                        <div className="space-y-1 min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <Link
                              to={`/${owner}/${repoSlug}/actions/runs/${run.id}`}
                              className="font-medium text-white hover:text-forge-accent text-sm leading-snug transition break-words"
                            >
                              {run.commit_message
                                ? run.commit_message.split('\n')[0]
                                : run.workflow_name}
                            </Link>

                            <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-forge-bg text-forge-accent border border-forge-border">
                              {run.workflow_name}
                            </span>
                          </div>

                          <div className="flex flex-wrap items-center gap-2 text-xs text-forge-muted">
                            <span className="inline-flex items-center gap-1 font-mono text-[11px] text-white">
                              <GitBranch className="w-3 h-3 text-forge-muted" />
                              {run.branch}
                            </span>
                            <span>•</span>
                            <span className="font-mono text-[11px] text-forge-muted flex items-center gap-1">
                              <GitCommitIcon className="w-3 h-3" />
                              {run.commit_sha.slice(0, 7)}
                            </span>
                            <span>•</span>
                            <span>
                              {run.trigger_event} by{' '}
                              <strong className="text-white">
                                {run.trigger_user?.username || 'system'}
                              </strong>
                            </span>
                          </div>
                        </div>
                      </div>

                      <div className="text-right text-xs font-mono shrink-0 space-y-1">
                        <p className="text-forge-muted">
                          {formatDuration(run.duration_ms, run.status)}
                        </p>
                        <p className="text-[11px] text-forge-muted/60">
                          {new Date(run.created_at).toLocaleTimeString([], {
                            hour: '2-digit',
                            minute: '2-digit',
                          })}
                        </p>
                      </div>
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
