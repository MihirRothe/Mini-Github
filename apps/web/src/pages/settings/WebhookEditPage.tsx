import React, { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { Repository, Webhook, WebhookDelivery } from '../../types';
import { RepoHeader } from '../../components/repo/RepoHeader';
import {
  Webhook as WebhookIcon,
  CheckCircle2,
  XCircle,
  Loader2,
  AlertCircle,
  Trash2,
  Send,
  Eye,
  EyeOff,
  RotateCcw,
  ArrowLeft,
  ChevronRight,
  ChevronDown
} from 'lucide-react';

export const WebhookEditPage: React.FC = () => {
  const { owner, repo: repoSlug, hookId } = useParams<{
    owner: string;
    repo: string;
    hookId: string;
  }>();
  const navigate = useNavigate();
  const isNew = !hookId || hookId === 'new';

  const [repo, setRepo] = useState<Repository | null>(null);
  const [hook, setHook] = useState<Webhook | null>(null);
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);
  const [selectedDelivery, setSelectedDelivery] = useState<WebhookDelivery | null>(null);

  // Form State
  const [url, setUrl] = useState('');
  const [contentType, setContentType] = useState('application/json');
  const [secret, setSecret] = useState('');
  const [showSecret, setShowSecret] = useState(false);
  const [sslVerification, setSslVerification] = useState(true);
  const [isActive, setIsActive] = useState(true);

  const [eventChoice, setEventChoice] = useState<'push' | 'all' | 'custom'>('push');
  const [customEvents, setCustomEvents] = useState<Record<string, boolean>>({
    push: true,
    pull_request: false,
    issues: false,
    issue_comment: false,
  });

  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [redelivering, setRedelivering] = useState(false);
  const [loading, setLoading] = useState(!isNew);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

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

  // 2. Fetch Existing Webhook and Deliveries
  const fetchHookData = async () => {
    if (isNew || !owner || !repoSlug || !hookId) return;
    setLoading(true);

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}`, {
        credentials: 'include',
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to load webhook');
      }
      const h: Webhook = data.webhook;
      setHook(h);
      setUrl(h.url);
      setContentType(h.content_type);
      setSslVerification(h.ssl_verification);
      setIsActive(h.is_active);

      if (h.events.includes('*')) {
        setEventChoice('all');
      } else if (h.events.length === 1 && h.events[0] === 'push') {
        setEventChoice('push');
      } else {
        setEventChoice('custom');
        const customMap: Record<string, boolean> = {};
        h.events.forEach((e) => {
          customMap[e] = true;
        });
        setCustomEvents((prev) => ({ ...prev, ...customMap }));
      }

      // Fetch Deliveries
      fetchDeliveries();
    } catch (err: any) {
      setError(err.message || 'Error loading webhook');
    } finally {
      setLoading(false);
    }
  };

  const fetchDeliveries = async () => {
    if (isNew || !owner || !repoSlug || !hookId) return;

    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}/deliveries`,
        { credentials: 'include' }
      );
      const data = await res.json();
      if (res.ok && data.deliveries) {
        setDeliveries(data.deliveries);
        if (data.deliveries.length > 0 && !selectedDelivery) {
          setSelectedDelivery(data.deliveries[0]);
        }
      }
    } catch (err) {}
  };

  useEffect(() => {
    fetchHookData();
  }, [owner, repoSlug, hookId, isNew]);

  // Compute events to save
  const resolveEvents = (): string[] => {
    if (eventChoice === 'push') return ['push'];
    if (eventChoice === 'all') return ['*'];
    const chosen = Object.entries(customEvents)
      .filter(([_, v]) => v)
      .map(([k]) => k);
    return chosen.length > 0 ? chosen : ['push'];
  };

  // 3. Save / Update Webhook
  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!owner || !repoSlug) return;

    setSaving(true);
    setError(null);
    setSuccessMsg(null);

    const bodyData = {
      url: url.trim(),
      content_type: contentType,
      secret: secret.trim() || undefined,
      events: resolveEvents(),
      is_active: isActive,
      ssl_verification: sslVerification,
    };

    try {
      const endpoint = isNew
        ? `/api/v1/repos/${owner}/${repoSlug}/settings/hooks`
        : `/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}`;
      const method = isNew ? 'POST' : 'PATCH';

      const res = await fetch(endpoint, {
        method,
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(bodyData),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to save webhook');
      }

      if (isNew) {
        navigate(`/${owner}/${repoSlug}/settings/hooks/${data.webhook.id}`);
      } else {
        setSuccessMsg('Webhook updated successfully.');
        setHook(data.webhook);
        await fetchDeliveries();
      }
    } catch (err: any) {
      setError(err.message || 'Error saving webhook');
    } finally {
      setSaving(false);
    }
  };

  // 4. Test Ping
  const handleTestPing = async () => {
    if (isNew || !owner || !repoSlug || !hookId) return;

    setTesting(true);
    setError(null);
    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}/tests`,
        { method: 'POST', credentials: 'include' }
      );
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Ping failed');
      }
      setSuccessMsg('Test ping delivered! See delivery details below.');
      await fetchDeliveries();
    } catch (err: any) {
      setError(err.message || 'Test ping failed');
    } finally {
      setTesting(false);
    }
  };

  // 5. Redeliver
  const handleRedeliver = async (deliveryId: string) => {
    if (isNew || !owner || !repoSlug || !hookId) return;

    setRedelivering(true);
    try {
      const res = await fetch(
        `/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}/deliveries/${deliveryId}/redeliver`,
        { method: 'POST', credentials: 'include' }
      );
      if (res.ok) {
        await fetchDeliveries();
      }
    } catch (err) {} finally {
      setRedelivering(false);
    }
  };

  // 6. Delete Webhook
  const handleDelete = async () => {
    if (isNew || !owner || !repoSlug || !hookId) return;
    if (!window.confirm('Are you sure you want to delete this webhook?')) return;

    try {
      const res = await fetch(`/api/v1/repos/${owner}/${repoSlug}/settings/hooks/${hookId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (res.ok) {
        navigate(`/${owner}/${repoSlug}/settings/hooks`);
      }
    } catch (err) {}
  };

  if (loading) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading webhook configuration...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-forge-bg text-forge-text pb-16">
      {repo && <RepoHeader repo={repo} activeTab="settings" />}

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6 space-y-6">
        {/* Breadcrumb Back */}
        <div className="flex items-center gap-2 text-xs text-forge-muted">
          <Link to={`/${owner}/${repoSlug}/settings/hooks`} className="hover:text-white flex items-center gap-1">
            <ArrowLeft className="w-3.5 h-3.5" />
            <span>Webhooks</span>
          </Link>
          <span>/</span>
          <span className="text-white font-medium">{isNew ? 'New webhook' : 'Manage webhook'}</span>
        </div>

        <div className="max-w-4xl space-y-6">
          <div className="card p-6 border border-forge-border space-y-6">
            <div className="flex items-center justify-between border-b border-forge-border pb-4">
              <h1 className="text-lg font-bold text-white flex items-center gap-2">
                <WebhookIcon className="w-5 h-5 text-forge-accent" />
                <span>{isNew ? 'Add webhook' : 'Webhook settings'}</span>
              </h1>

              {!isNew && (
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={handleTestPing}
                    disabled={testing}
                    className="btn-secondary text-xs flex items-center gap-1.5"
                  >
                    {testing ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Send className="w-3.5 h-3.5" />}
                    <span>Test ping</span>
                  </button>
                  <button
                    type="button"
                    onClick={handleDelete}
                    className="p-2 text-forge-muted hover:text-rose-400 transition rounded-lg hover:bg-forge-card"
                    title="Delete webhook"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              )}
            </div>

            {/* Error / Success Notifications */}
            {error && (
              <div className="p-4 bg-rose-500/10 border border-rose-500/30 rounded-lg flex items-center gap-3 text-rose-400 text-sm">
                <AlertCircle className="w-5 h-5 shrink-0" />
                <span>{error}</span>
              </div>
            )}
            {successMsg && (
              <div className="p-4 bg-emerald-500/10 border border-emerald-500/30 rounded-lg flex items-center gap-3 text-emerald-300 text-sm">
                <CheckCircle2 className="w-5 h-5 shrink-0 text-emerald-400" />
                <span>{successMsg}</span>
              </div>
            )}

            {/* Webhook Configuration Form */}
            <form onSubmit={handleSave} className="space-y-5">
              {/* Payload URL */}
              <div>
                <label className="block text-xs font-semibold text-white mb-1.5">
                  Payload URL <span className="text-rose-400">*</span>
                </label>
                <input
                  type="url"
                  required
                  placeholder="https://example.com/postreceive"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  className="w-full px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-xs text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent font-mono"
                />
              </div>

              {/* Content Type */}
              <div>
                <label className="block text-xs font-semibold text-white mb-1.5">
                  Content type
                </label>
                <select
                  value={contentType}
                  onChange={(e) => setContentType(e.target.value)}
                  className="w-full sm:w-80 px-3 py-2 bg-forge-bg border border-forge-border rounded-lg text-xs text-white focus:outline-none focus:border-forge-accent"
                >
                  <option value="application/json">application/json</option>
                  <option value="application/x-www-form-urlencoded">application/x-www-form-urlencoded</option>
                </select>
              </div>

              {/* Secret Key */}
              <div>
                <label className="block text-xs font-semibold text-white mb-1.5">
                  Secret
                </label>
                <div className="relative w-full sm:w-96">
                  <input
                    type={showSecret ? 'text' : 'password'}
                    placeholder={hook?.has_secret ? '••••••••••••••••' : 'Optional HMAC-SHA256 secret'}
                    value={secret}
                    onChange={(e) => setSecret(e.target.value)}
                    className="w-full px-3 py-2 pr-9 bg-forge-bg border border-forge-border rounded-lg text-xs text-white placeholder-forge-muted focus:outline-none focus:border-forge-accent font-mono"
                  />
                  <button
                    type="button"
                    onClick={() => setShowSecret(!showSecret)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-forge-muted hover:text-white"
                  >
                    {showSecret ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                  </button>
                </div>
                <p className="text-[11px] text-forge-muted mt-1">
                  Used to generate an HMAC-SHA256 signature in the <code className="text-forge-accent">X-Forge-Signature-256</code> header.
                </p>
              </div>

              {/* SSL Verification */}
              <div className="pt-2">
                <label className="flex items-start gap-2.5 cursor-pointer text-xs select-none">
                  <input
                    type="checkbox"
                    checked={sslVerification}
                    onChange={(e) => setSslVerification(e.target.checked)}
                    className="mt-0.5 rounded border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                  />
                  <div>
                    <span className="text-white font-medium">Enable SSL verification</span>
                    <p className="text-forge-muted text-[11px] mt-0.5">
                      Verify SSL certificate authenticity when delivering payloads to HTTPS URLs.
                    </p>
                  </div>
                </label>
              </div>

              {/* Trigger Events */}
              <div className="pt-2 border-t border-forge-border/60 space-y-3">
                <label className="block text-xs font-semibold text-white">
                  Which events would you like to trigger this webhook?
                </label>

                <div className="space-y-2 text-xs">
                  <label className="flex items-center gap-2 cursor-pointer select-none">
                    <input
                      type="radio"
                      name="eventChoice"
                      value="push"
                      checked={eventChoice === 'push'}
                      onChange={() => setEventChoice('push')}
                      className="border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                    />
                    <span className="text-white">Just the <strong className="text-forge-accent">push</strong> event.</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer select-none">
                    <input
                      type="radio"
                      name="eventChoice"
                      value="all"
                      checked={eventChoice === 'all'}
                      onChange={() => setEventChoice('all')}
                      className="border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                    />
                    <span className="text-white">Send me <strong>everything</strong> (wildcard *).</span>
                  </label>

                  <label className="flex items-center gap-2 cursor-pointer select-none">
                    <input
                      type="radio"
                      name="eventChoice"
                      value="custom"
                      checked={eventChoice === 'custom'}
                      onChange={() => setEventChoice('custom')}
                      className="border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                    />
                    <span className="text-white">Let me select individual events.</span>
                  </label>
                </div>

                {eventChoice === 'custom' && (
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 pl-6 pt-1 text-xs text-forge-muted">
                    {['push', 'pull_request', 'issues', 'issue_comment'].map((ev) => (
                      <label key={ev} className="flex items-center gap-2 cursor-pointer select-none">
                        <input
                          type="checkbox"
                          checked={!!customEvents[ev]}
                          onChange={(e) =>
                            setCustomEvents((prev) => ({ ...prev, [ev]: e.target.checked }))
                          }
                          className="rounded border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                        />
                        <span className="text-white capitalize">{ev.replace('_', ' ')}</span>
                      </label>
                    ))}
                  </div>
                )}
              </div>

              {/* Active Toggle */}
              <div className="pt-2 border-t border-forge-border/60">
                <label className="flex items-center gap-2 cursor-pointer text-xs select-none">
                  <input
                    type="checkbox"
                    checked={isActive}
                    onChange={(e) => setIsActive(e.target.checked)}
                    className="rounded border-forge-border bg-forge-bg text-forge-accent focus:ring-0"
                  />
                  <span className="text-white font-medium">Active</span>
                </label>
                <p className="text-[11px] text-forge-muted pl-6 mt-0.5">
                  We will deliver event details when this hook is triggered.
                </p>
              </div>

              <div className="pt-3 flex justify-end">
                <button
                  type="submit"
                  disabled={saving}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  {saving && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                  <span>{isNew ? 'Add webhook' : 'Update webhook'}</span>
                </button>
              </div>
            </form>
          </div>

          {/* Recent Deliveries Table & Details */}
          {!isNew && (
            <div className="card overflow-hidden border border-forge-border space-y-4">
              <div className="px-4 py-3 bg-forge-card/80 border-b border-forge-border flex items-center justify-between text-xs font-semibold text-white">
                <span>Recent Deliveries ({deliveries.length})</span>
                <button
                  type="button"
                  onClick={fetchDeliveries}
                  className="text-forge-muted hover:text-white transition font-normal"
                >
                  Refresh
                </button>
              </div>

              {deliveries.length === 0 ? (
                <div className="p-8 text-center text-xs text-forge-muted">
                  No deliveries recorded yet. Click "Test ping" to test connection.
                </div>
              ) : (
                <div className="divide-y divide-forge-border/60">
                  {deliveries.map((del) => {
                    const isExpanded = selectedDelivery?.id === del.id;
                    return (
                      <div key={del.id} className="text-xs">
                        {/* Delivery Row */}
                        <div
                          onClick={() => setSelectedDelivery(isExpanded ? null : del)}
                          className="p-3.5 hover:bg-forge-card/40 cursor-pointer flex items-center justify-between gap-4 transition select-none"
                        >
                          <div className="flex items-center gap-3 min-w-0">
                            {isExpanded ? (
                              <ChevronDown className="w-3.5 h-3.5 text-forge-muted shrink-0" />
                            ) : (
                              <ChevronRight className="w-3.5 h-3.5 text-forge-muted shrink-0" />
                            )}

                            {del.is_success ? (
                              <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                            ) : (
                              <XCircle className="w-4 h-4 text-rose-400 shrink-0" />
                            )}

                            <span className="font-mono font-semibold text-[11px] px-1.5 py-0.5 rounded bg-forge-bg border border-forge-border text-white">
                              {del.response_status_code || 'ERR'}
                            </span>

                            <span className="font-mono text-xs text-forge-accent truncate">
                              {del.event}
                            </span>
                          </div>

                          <div className="flex items-center gap-3 text-forge-muted font-mono text-[11px] shrink-0">
                            <span>{del.duration_ms}ms</span>
                            <span>{new Date(del.delivered_at).toLocaleTimeString()}</span>
                          </div>
                        </div>

                        {/* Expanded Details */}
                        {isExpanded && (
                          <div className="p-4 bg-forge-bg/80 border-t border-forge-border/60 space-y-4 font-mono text-xs">
                            <div className="flex justify-between items-center">
                              <span className="text-forge-muted">Delivery ID: {del.id}</span>
                              <button
                                type="button"
                                onClick={() => handleRedeliver(del.id)}
                                disabled={redelivering}
                                className="btn-secondary text-[11px] flex items-center gap-1.5"
                              >
                                {redelivering ? (
                                  <Loader2 className="w-3 h-3 animate-spin" />
                                ) : (
                                  <RotateCcw className="w-3 h-3" />
                                )}
                                <span>Redeliver</span>
                              </button>
                            </div>

                            {/* Request Payload */}
                            <div className="space-y-1">
                              <span className="text-[11px] text-white font-semibold block">Request Payload:</span>
                              <pre className="p-3 bg-forge-card border border-forge-border rounded-lg overflow-x-auto text-[11px] text-[#c9d1d9] max-h-48">
                                {JSON.stringify(del.request_payload, null, 2)}
                              </pre>
                            </div>

                            {/* Response Body */}
                            {del.response_body && (
                              <div className="space-y-1">
                                <span className="text-[11px] text-white font-semibold block">Response Body:</span>
                                <pre className="p-3 bg-forge-card border border-forge-border rounded-lg overflow-x-auto text-[11px] text-[#c9d1d9] max-h-36">
                                  {del.response_body}
                                </pre>
                              </div>
                            )}

                            {del.error_message && (
                              <div className="p-2.5 bg-rose-500/10 border border-rose-500/30 rounded text-rose-400 text-xs">
                                Error: {del.error_message}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
