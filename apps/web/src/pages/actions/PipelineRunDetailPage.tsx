import React, { useEffect, useState, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, PipelineRunDetail, PipelineJob, JobLog } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import {
  CheckCircle2,
  XCircle,
  Loader2,
  Clock,
  Ban,
  GitBranch,
  GitCommit as GitCommitIcon,
  RotateCcw,
  Square,
  Terminal,
  Copy,
  Check,
  ChevronRight,
  ArrowLeft,
  AlertCircle
} from 'lucide-react';

export const PipelineRunDetailPage: React.FC = () => {
  const { owner, repo: repoSlug, runId } = useParams<{
    owner: string;
    repo: string;
    runId: string;
  }>();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [run, setRun] = useState<PipelineRunDetail | null>(null);
  const [selectedJob, setSelectedJob] = useState<PipelineJob | null>(null);
  const [logs, setLogs] = useState<JobLog[]>([]);
  const [selectedStepId, setSelectedStepId] = useState<string | null>(null);

  const [autoScroll, setAutoScroll] = useState(true);
  const [copied, setCopied] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [rerunning, setRerunning] = useState(false);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const terminalEndRef = useRef<HTMLDivElement>(null);

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

  // 2. Fetch Pipeline Run Details
  const fetchRun = async () => {
    if (!owner || !repoSlug || !runId) return;

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/actions/runs/${runId}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load run details');
      }
      setRun(data.run);

      // Auto-select first job if none selected
      if (data.run?.jobs && data.run.jobs.length > 0) {
        setSelectedJob((prev) => {
          if (!prev) return data.run.jobs[0];
          const found = data.run.jobs.find((j: PipelineJob) => j.id === prev.id);
          return found || data.run.jobs[0];
        });
      }
    } catch (err: any) {
      setError(err.message || 'Error fetching run');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRun();
  }, [owner, repoSlug, runId]);

  // 3. Fetch Logs for Selected Job
  const fetchLogs = async (jobId: string, after: number) => {
    if (!owner || !repoSlug) return;

    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repoSlug}/actions/jobs/${jobId}/logs?after=${after}`,
        { credentials: 'include' }
      );
      const data = await res.json();
      if (res.ok && data.logs) {
        if (after === 0) {
          setLogs(data.logs);
        } else if (data.logs.length > 0) {
          setLogs((prev) => [...prev, ...data.logs]);
        }
      }
    } catch (err) {}
  };

  useEffect(() => {
    if (!selectedJob) return;
    setLogs([]);
    fetchLogs(selectedJob.id, 0);
  }, [selectedJob?.id]);

  // 4. Polling for Live Logs and Status Updates if Running
  useEffect(() => {
    if (!run || !selectedJob) return;
    const isLive = run.status === 'running' || run.status === 'queued';
    if (!isLive) return;

    const interval = setInterval(() => {
      fetchRun();
      const lastLine = logs.length > 0 ? logs[logs.length - 1].line_number : 0;
      fetchLogs(selectedJob.id, lastLine);
    }, 1500);

    return () => clearInterval(interval);
  }, [run?.status, selectedJob?.id, logs.length]);

  // Auto scroll to bottom of terminal
  useEffect(() => {
    if (autoScroll && terminalEndRef.current) {
      terminalEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  // 5. Cancel Pipeline Run
  const handleCancel = async () => {
    if (!owner || !repoSlug || !runId) return;
    setCancelling(true);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/actions/runs/${runId}/cancel`, {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        const d = await res.json();
        throw new Error(d.error?.message || 'Cancel failed');
      }
      await fetchRun();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setCancelling(false);
    }
  };

  // 6. Rerun Pipeline
  const handleRerun = async () => {
    if (!owner || !repoSlug || !runId) return;
    setRerunning(true);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/actions/runs/${runId}/rerun`, {
        method: 'POST',
        credentials: 'include',
      });
      const d = await res.json();
      if (!res.ok) {
        throw new Error(d.error?.message || 'Rerun failed');
      }
      window.location.href = `/${owner}/${repoSlug}/actions/runs/${d.run.id}`;
    } catch (err: any) {
      setError(err.message);
      setRerunning(false);
    }
  };

  // Copy Logs
  const handleCopyLogs = () => {
    const text = logs.map((l) => l.content).join('\n');
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
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
        return <Clock className="w-5 h-5 text-forge-muted shrink-0" />;
    }
  };

  if (loading && !run) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading pipeline run details...</p>
      </div>
    );
  }

  if (!run) {
    return (
      <div className="max-w-4xl mx-auto py-16 px-4 text-center">
        <AlertCircle className="w-10 h-10 text-rose-400 mx-auto mb-3" />
        <h2 className="text-lg font-semibold text-white">Pipeline run not found</h2>
        <Link
          to={`/${owner}/${repoSlug}/actions`}
          className="btn-primary text-xs inline-flex items-center gap-1.5 mt-4"
        >
          Back to all runs
        </Link>
      </div>
    );
  }

  const isLive = run.status === 'running' || run.status === 'queued';
  const displayedLogs = selectedStepId
    ? logs.filter((l) => l.step_id === selectedStepId)
    : logs;

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-16">
      {repo && <RepoHeader repo={repo} activeTab="actions" />}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        {/* Breadcrumb back */}
        <div className="flex items-center gap-2 text-xs text-forge-muted">
          <Link
            to={`/${owner}/${repoSlug}/actions`}
            className="hover:text-white flex items-center gap-1"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            <span>Actions</span>
          </Link>
          <span>/</span>
          <span className="text-white font-medium">{run.workflow_name}</span>
        </div>

        {/* Run Header Banner */}
        <div className="card p-5 border border-forge-border space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-start gap-3.5 min-w-0">
              <div className="mt-1">{getStatusIcon(run.status)}</div>

              <div className="space-y-1 min-w-0">
                <h1 className="text-lg sm:text-xl font-bold text-white leading-tight break-words">
                  {run.commit_message ? run.commit_message.split('\n')[0] : run.workflow_name}
                </h1>

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
                    Workflow: <strong className="text-forge-accent font-mono">{run.workflow_file}</strong>
                  </span>
                  <span>•</span>
                  <span>{run.trigger_event}</span>
                </div>
              </div>
            </div>

            {/* Actions: Cancel or Re-run */}
            <div className="flex items-center gap-2 shrink-0">
              {isLive ? (
                <button
                  type="button"
                  onClick={handleCancel}
                  disabled={cancelling}
                  className="btn-secondary text-xs flex items-center gap-1.5 text-rose-400 hover:text-rose-300"
                >
                  {cancelling ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Square className="w-3.5 h-3.5" />
                  )}
                  <span>Cancel run</span>
                </button>
              ) : (
                <button
                  type="button"
                  onClick={handleRerun}
                  disabled={rerunning}
                  className="btn-secondary text-xs flex items-center gap-1.5"
                >
                  {rerunning ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <RotateCcw className="w-3.5 h-3.5" />
                  )}
                  <span>Re-run all jobs</span>
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
            <AlertCircle className="w-5 h-5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Two Column Layout: Jobs & Steps sidebar + Real-time Terminal */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Jobs & Steps Sidebar */}
          <div className="space-y-4">
            <div className="card p-4 space-y-3 border border-forge-border">
              <h2 className="text-xs font-semibold text-white uppercase tracking-wider">
                Jobs ({(run.jobs || []).length})
              </h2>

              <div className="space-y-2">
                {run.jobs?.map((job) => {
                  const isSelected = selectedJob?.id === job.id;
                  return (
                    <div
                      key={job.id}
                      onClick={() => {
                        setSelectedJob(job);
                        setSelectedStepId(null);
                      }}
                      className={`p-3 rounded-lg border cursor-pointer select-none transition space-y-2 ${
                        isSelected
                          ? 'bg-forge-accent/15 border-forge-accent/40 text-white'
                          : 'bg-forge-bg/60 border-forge-border/80 text-forge-muted hover:text-white hover:bg-forge-card'
                      }`}
                    >
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-2 min-w-0">
                          {getStatusIcon(job.status)}
                          <span className="font-semibold text-xs truncate">{job.name}</span>
                        </div>
                        <span className="text-[11px] font-mono text-forge-muted shrink-0">
                          {job.duration_ms ? `${Math.round(job.duration_ms / 1000)}s` : ''}
                        </span>
                      </div>

                      {/* Steps under this job */}
                      {job.steps && job.steps.length > 0 && isSelected && (
                        <div className="pt-2 border-t border-forge-border/40 space-y-1">
                          {job.steps.map((step) => {
                            const isStepActive = selectedStepId === step.id;
                            return (
                              <button
                                key={step.id}
                                type="button"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setSelectedStepId(isStepActive ? null : step.id);
                                }}
                                className={`w-full text-left px-2 py-1 rounded text-[11px] flex items-center justify-between gap-2 transition ${
                                  isStepActive
                                    ? 'bg-forge-accent/25 text-white font-medium'
                                    : 'text-forge-muted hover:text-white hover:bg-forge-bg'
                                }`}
                              >
                                <div className="flex items-center gap-1.5 min-w-0">
                                  <ChevronRight
                                    className={`w-3 h-3 text-forge-muted transition-transform ${
                                      isStepActive ? 'rotate-90' : ''
                                    }`}
                                  />
                                  <span className="truncate">{step.name}</span>
                                </div>
                                <span className="font-mono text-[10px] text-forge-muted/80">
                                  {step.duration_ms ? `${Math.round(step.duration_ms / 1000)}s` : ''}
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          {/* Live Terminal Log Viewer */}
          <div className="lg:col-span-2">
            <div className="rounded-xl border border-forge-border overflow-hidden bg-[#0d1117] flex flex-col h-[650px] shadow-2xl">
              {/* Terminal Toolbar */}
              <div className="px-4 py-2.5 bg-[#161b22] border-b border-forge-border flex items-center justify-between gap-4 text-xs select-none">
                <div className="flex items-center gap-2 text-forge-muted">
                  <Terminal className="w-4 h-4 text-forge-accent" />
                  <span className="font-mono text-white font-medium">
                    {selectedJob?.name || 'Console Log'}
                  </span>
                  {selectedStepId && (
                    <span className="text-forge-accent text-[11px]">
                      (filtered by step)
                    </span>
                  )}
                </div>

                <div className="flex items-center gap-3">
                  <label className="flex items-center gap-1.5 cursor-pointer text-[11px] text-forge-muted hover:text-white">
                    <input
                      type="checkbox"
                      checked={autoScroll}
                      onChange={(e) => setAutoScroll(e.target.checked)}
                      className="rounded border-forge-border bg-forge-bg text-forge-accent"
                    />
                    <span>Auto-scroll</span>
                  </label>

                  <button
                    type="button"
                    onClick={handleCopyLogs}
                    className="flex items-center gap-1 text-[11px] text-forge-muted hover:text-white px-2 py-1 rounded hover:bg-forge-bg transition"
                  >
                    {copied ? (
                      <>
                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                        <span className="text-emerald-400">Copied</span>
                      </>
                    ) : (
                      <>
                        <Copy className="w-3.5 h-3.5" />
                        <span>Copy logs</span>
                      </>
                    )}
                  </button>
                </div>
              </div>

              {/* Terminal Screen */}
              <div className="flex-1 p-4 overflow-y-auto font-mono text-xs text-[#c9d1d9] space-y-0.5 leading-5 select-text">
                {displayedLogs.length === 0 ? (
                  <div className="h-full flex flex-col items-center justify-center text-forge-muted/60 space-y-2">
                    <Loader2 className="w-6 h-6 animate-spin text-forge-accent" />
                    <p className="text-xs">Waiting for console output...</p>
                  </div>
                ) : (
                  displayedLogs.map((log) => {
                    const isSystem = log.stream === 'system';
                    const isStderr = log.stream === 'stderr';

                    let colorClass = 'text-[#c9d1d9]';
                    if (isSystem) {
                      colorClass = 'text-cyan-400 font-semibold';
                    } else if (isStderr) {
                      colorClass = 'text-rose-400';
                    }

                    return (
                      <div
                        key={log.id}
                        className="flex items-start hover:bg-white/[0.02] -mx-2 px-2 rounded"
                      >
                        <span className="w-10 select-none text-right pr-4 text-forge-muted/40 shrink-0 text-[11px]">
                          {log.line_number}
                        </span>
                        <span className={`whitespace-pre-wrap break-all ${colorClass}`}>
                          {log.content}
                        </span>
                      </div>
                    );
                  })
                )}
                <div ref={terminalEndRef} />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
