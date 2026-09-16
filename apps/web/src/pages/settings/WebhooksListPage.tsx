import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { Repository, Webhook } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import {
  Webhook as WebhookIcon,
  Plus,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
  Trash2,
  Edit2
} from 'lucide-react';

export const WebhooksListPage: React.FC = () => {
  const { owner, repo: repoSlug } = useParams<{ owner: string; repo: string }>();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
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

  // 2. Fetch Webhooks
  const fetchWebhooks = async () => {
    if (!owner || !repoSlug) return;
    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/settings/hooks`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load webhooks');
      }
      setWebhooks(data.webhooks || []);
    } catch (err: any) {
      setError(err.message || 'Error fetching webhooks');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchWebhooks();
  }, [owner, repoSlug]);

  // 3. Delete Webhook
  const handleDelete = async (hookId: string) => {
    if (!owner || !repoSlug) return;
    if (!window.confirm('Are you sure you want to delete this webhook?')) return;

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        await fetchWebhooks();
      }
    } catch (err) {}
  };

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-16">
      {repo && <RepoHeader repo={repo} activeTab="settings" />}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Settings Sidebar */}
          <div className="space-y-2">
            <div className="card p-3 border border-forge-border">
              <h2 className="text-xs font-semibold text-white uppercase tracking-wider px-3 py-2">
                Settings
              </h2>
              <div className="space-y-1 text-xs">
                <Link
                  to={`/${owner}/${repoSlug}/settings/hooks`}
                  className="flex items-center gap-2 px-3 py-2 rounded-lg bg-forge-accent/15 text-white font-medium border border-forge-accent/30"
                >
                  <WebhookIcon className="w-4 h-4 text-forge-accent" />
                  <span>Webhooks</span>
                </Link>
              </div>
            </div>
          </div>

          {/* Main Webhooks List Content */}
          <div className="lg:col-span-3 space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div>
                <h1 className="text-xl font-bold text-white flex items-center gap-2">
                  <WebhookIcon className="w-5 h-5 text-forge-accent" />
                  <span>Webhooks</span>
                </h1>
                <p className="text-xs text-forge-muted mt-1">
                  Webhooks allow external services to be notified when certain events happen on ForgeHub.
                </p>
              </div>

              <Link
                to={`/${owner}/${repoSlug}/settings/hooks/new`}
                className="btn-primary text-xs flex items-center gap-1.5 shadow-lg shadow-forge-accent/10 whitespace-nowrap"
              >
                <Plus className="w-3.5 h-3.5" />
                <span>Add webhook</span>
              </Link>
            </div>

            {/* Error Alert */}
            {error && (
              <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
                <AlertCircle className="w-5 h-5 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            {/* Webhooks Card List */}
            <div className="card overflow-hidden border border-forge-border">
              <div className="px-4 py-3 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs font-medium text-white">
                <span>Configured Webhooks ({webhooks.length})</span>
              </div>

              {loading ? (
                <div className="p-12 text-center text-forge-muted">
                  <Loader2 className="w-6 h-6 text-forge-accent animate-spin mx-auto mb-2" />
                  <p className="text-xs">Fetching webhooks...</p>
                </div>
              ) : webhooks.length === 0 ? (
                <div className="p-12 text-center text-forge-muted space-y-3">
                  <WebhookIcon className="w-10 h-10 mx-auto text-forge-muted/40" />
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-white">No webhooks yet</p>
                    <p className="text-xs text-forge-muted max-w-sm mx-auto">
                      Send real-time HTTP POST payloads to CI/CD platforms, Discord, Slack, or custom automation endpoints.
                    </p>
                  </div>
                  <Link
                    to={`/${owner}/${repoSlug}/settings/hooks/new`}
                    className="btn-primary text-xs inline-flex items-center gap-1.5 mt-2"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Add your first webhook
                  </Link>
                </div>
              ) : (
                <div className="divide-y divide-forge-border/60">
                  {webhooks.map((hook) => {
                    const isSuccess = hook.last_status && hook.last_status >= 200 && hook.last_status < 300;
                    const isFailed = hook.last_status && (hook.last_status < 200 || hook.last_status >= 300);

                    return (
                      <div
                        key={hook.id}
                        className="p-4 hover:bg-forge-card/40 transition flex items-center justify-between gap-4"
                      >
                        <div className="flex items-start gap-3.5 min-w-0">
                          <div
                            className="mt-1"
                            title={
                              isSuccess
                                ? 'Last delivery succeeded'
                                : isFailed
                                ? `Last delivery failed (${hook.last_status})`
                                : 'No deliveries yet'
                            }
                          >
                            {isSuccess ? (
                              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                            ) : isFailed ? (
                              <XCircle className="w-4 h-4 text-rose-400" />
                            ) : (
                              <Clock className="w-4 h-4 text-forge-muted" />
                            )}
                          </div>

                          <div className="space-y-1 min-w-0">
                            <Link
                              to={`/${owner}/${repoSlug}/settings/hooks/${hook.id}`}
                              className="font-mono text-xs text-white hover:text-forge-accent font-medium truncate block max-w-md"
                            >
                              {hook.url}
                            </Link>

                            <div className="flex flex-wrap items-center gap-2 text-xs text-forge-muted">
                              <span className="font-mono text-[11px] bg-forge-bg px-2 py-0.5 rounded border border-forge-border text-forge-accent">
                                {hook.events.includes('*') ? 'All events' : hook.events.join(', ')}
                              </span>

                              {hook.has_secret && (
                                <span className="text-[10px] px-1.5 py-0.2 rounded bg-forge-bg text-emerald-400 border border-emerald-500/30 font-mono">
                                  HMAC-SHA256
                                </span>
                              )}

                              {!hook.is_active && (
                                <span className="text-[10px] px-1.5 py-0.2 rounded bg-forge-bg text-rose-400 border border-rose-500/30">
                                  Disabled
                                </span>
                              )}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 shrink-0">
                          <Link
                            to={`/${owner}/${repoSlug}/settings/hooks/${hook.id}`}
                            className="p-1.5 rounded text-forge-muted hover:text-white hover:bg-forge-card transition"
                            title="Edit webhook"
                          >
                            <Edit2 className="w-3.5 h-3.5" />
                          </Link>
                          <button
                            type="button"
                            onClick={() => handleDelete(hook.id)}
                            className="p-1.5 rounded text-forge-muted hover:text-rose-400 hover:bg-forge-card transition"
                            title="Delete webhook"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
