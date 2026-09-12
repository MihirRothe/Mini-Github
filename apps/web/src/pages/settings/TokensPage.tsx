import React, { useState } from 'react';
import { Navigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import { Key, Plus, Trash2, Copy, Check, X, Shield, User } from 'lucide-react';
import { APIToken } from '../../types';

export const TokensPage: React.FC = () => {
  const { user, isAuthenticated, isLoading } = useAuth();
  const queryClient = useQueryClient();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [tokenName, setTokenName] = useState('');
  const [expiresInDays, setExpiresInDays] = useState(30);
  const [selectedScopes, setSelectedScopes] = useState<string[]>(['repo:read', 'repo:write']);

  const [newToken, setNewToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const { data: tokens = [], isLoading: isLoadingTokens } = useQuery<APIToken[]>({
    queryKey: ['personal-access-tokens'],
    queryFn: async () => {
      const res = await fetch('/api/v1/tokens');
      if (!res.ok) throw new Error('Failed to load tokens');
      const data = await res.json();
      return data.tokens;
    },
    enabled: !!user,
  });

  const createTokenMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/tokens', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: tokenName,
          scopes: selectedScopes,
          expires_in_days: expiresInDays,
        }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error?.message || 'Failed to generate token');
      return data.token;
    },
    onSuccess: (tokenData) => {
      queryClient.invalidateQueries({ queryKey: ['personal-access-tokens'] });
      setNewToken(tokenData.token);
      setIsModalOpen(false);
      setTokenName('');
    },
    onError: (err: any) => {
      setErrorMsg(err.message || 'Error generating token');
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (id: string) => {
      const res = await fetch(`/api/v1/tokens/${id}`, { method: 'DELETE' });
      if (!res.ok) throw new Error('Failed to revoke token');
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['personal-access-tokens'] });
    },
  });

  if (isLoading) return <div className="py-12 text-center text-xs text-forge-muted">Loading settings...</div>;
  if (!isAuthenticated || !user) return <Navigate to="/login" replace />;

  const handleCopy = () => {
    if (newToken) {
      navigator.clipboard.writeText(newToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const availableScopes = [
    { id: 'repo:read', label: 'repo:read', desc: 'Read repository code, issues, and pull requests' },
    { id: 'repo:write', label: 'repo:write', desc: 'Push and pull code, create and edit issues/PRs' },
    { id: 'repo:admin', label: 'repo:admin', desc: 'Full administration of repositories' },
    { id: 'workflow:write', label: 'workflow:write', desc: 'Trigger and manage CI/CD workflow runs' },
    { id: 'user:read', label: 'user:read', desc: 'Access public and private user profile information' },
  ];

  const toggleScope = (scopeId: string) => {
    if (selectedScopes.includes(scopeId)) {
      setSelectedScopes(selectedScopes.filter(s => s !== scopeId));
    } else {
      setSelectedScopes([...selectedScopes, scopeId]);
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Settings Header */}
      <div className="pb-4 border-b border-forge-border">
        <h1 className="text-2xl font-bold tracking-tight text-white">Developer Settings</h1>
        <p className="text-xs text-forge-muted mt-1">
          Manage your personal profile information, security credentials, and access tokens.
        </p>
      </div>

      {/* Tabs */}
      <div className="flex items-center space-x-6 border-b border-forge-border pb-2 text-xs font-medium">
        <Link to="/settings/profile" className="flex items-center space-x-2 text-forge-muted hover:text-white transition-colors">
          <User className="w-4 h-4" />
          <span>Public Profile</span>
        </Link>
        <Link to="/settings/tokens" className="flex items-center space-x-2 pb-2 -mb-2 border-b-2 border-blue-500 text-white font-semibold">
          <Key className="w-4 h-4 text-blue-400" />
          <span>Personal Access Tokens</span>
        </Link>
      </div>

      {/* New Token Banner if just generated */}
      {newToken && (
        <div className="p-4 bg-emerald-500/10 border border-emerald-500/30 rounded-xl space-y-2 animate-fade-in">
          <div className="flex items-center space-x-2 text-emerald-400 font-semibold text-xs">
            <Shield className="w-4 h-4" />
            <span>Make sure to copy your personal access token now. You won't be able to see it again!</span>
          </div>
          <div className="flex items-center space-x-2">
            <code className="flex-1 px-3 py-2 bg-forge-card border border-forge-border rounded-lg text-xs font-mono text-emerald-300 select-all overflow-x-auto">
              {newToken}
            </code>
            <button
              onClick={handleCopy}
              className="py-2 px-3 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold flex items-center space-x-1.5 transition-colors shadow"
            >
              {copied ? <Check className="w-4 h-4" /> : <Copy className="w-4 h-4" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
        </div>
      )}

      {/* Token List Card */}
      <div className="glass-panel p-6 rounded-xl space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-sm font-semibold text-white">Active Access Tokens</h2>
            <p className="text-xs text-forge-muted">Tokens you have generated that can be used to access the ForgeHub API and Git over HTTPS.</p>
          </div>

          <button
            onClick={() => {
              setErrorMsg(null);
              setIsModalOpen(true);
            }}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-md transition-all"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Generate New Token</span>
          </button>
        </div>

        {isLoadingTokens ? (
          <div className="py-8 text-center text-xs text-forge-muted">Loading tokens...</div>
        ) : tokens.length === 0 ? (
          <div className="py-12 text-center space-y-2">
            <Key className="w-8 h-8 text-forge-muted mx-auto" />
            <p className="text-xs text-forge-muted">You haven't generated any personal access tokens yet.</p>
          </div>
        ) : (
          <div className="divide-y divide-forge-border/50">
            {tokens.map((token) => (
              <div key={token.id} className="py-3 flex items-center justify-between gap-4">
                <div className="space-y-1">
                  <div className="flex items-center space-x-2">
                    <span className="font-semibold text-xs text-white">{token.name}</span>
                    <span className="font-mono text-[10px] text-forge-muted bg-forge-card px-2 py-0.5 rounded border border-forge-border">
                      {token.token_prefix}...
                    </span>
                  </div>
                  <div className="flex flex-wrap gap-1.5 pt-0.5">
                    {token.scopes.map((scope, idx) => (
                      <span key={idx} className="text-[10px] font-mono text-blue-400 bg-blue-500/10 px-1.5 py-0.2 rounded border border-blue-500/20">
                        {scope}
                      </span>
                    ))}
                  </div>
                  <div className="text-[10px] text-forge-muted">
                    Created on {new Date(token.created_at).toLocaleDateString()}
                    {token.expires_at ? ` • Expires on ${new Date(token.expires_at).toLocaleDateString()}` : ' • Never expires'}
                  </div>
                </div>

                <button
                  onClick={() => revokeMutation.mutate(token.id)}
                  disabled={revokeMutation.isPending}
                  className="p-1.5 text-forge-muted hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
                  title="Revoke Token"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Modal: Generate Token */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
          <div className="w-full max-w-lg bg-forge-surface border border-forge-border rounded-xl shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between pb-2 border-b border-forge-border">
              <h3 className="text-base font-bold text-white">Generate Personal Access Token</h3>
              <button onClick={() => setIsModalOpen(false)} className="text-forge-muted hover:text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            {errorMsg && (
              <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-lg text-xs text-rose-400">
                {errorMsg}
              </div>
            )}

            <div className="space-y-3 text-xs">
              <div className="space-y-1">
                <label className="font-medium text-forge-muted">Token Name / Note</label>
                <input
                  type="text"
                  required
                  value={tokenName}
                  onChange={(e) => setTokenName(e.target.value)}
                  placeholder="e.g. CI runner on mac, home laptop CLI"
                  className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="space-y-1">
                <label className="font-medium text-forge-muted">Expiration</label>
                <select
                  value={expiresInDays}
                  onChange={(e) => setExpiresInDays(Number(e.target.value))}
                  className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
                >
                  <option value={30}>30 days</option>
                  <option value={60}>60 days</option>
                  <option value={90}>90 days</option>
                  <option value={0}>No expiration (not recommended)</option>
                </select>
              </div>

              <div className="space-y-2 pt-2">
                <label className="font-medium text-forge-muted">Select Scopes</label>
                <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                  {availableScopes.map((scope) => (
                    <label
                      key={scope.id}
                      className="flex items-start space-x-2.5 p-2 rounded-lg bg-forge-card border border-forge-border hover:border-forge-subtle cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={selectedScopes.includes(scope.id)}
                        onChange={() => toggleScope(scope.id)}
                        className="mt-0.5 rounded text-blue-600 focus:ring-0"
                      />
                      <div>
                        <div className="font-mono font-semibold text-forge-text text-[11px]">{scope.label}</div>
                        <div className="text-[10px] text-forge-muted">{scope.desc}</div>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end space-x-2 pt-3 border-t border-forge-border">
              <button
                type="button"
                onClick={() => setIsModalOpen(false)}
                className="px-3 py-1.5 rounded-lg text-xs text-forge-muted hover:text-white"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={createTokenMutation.isPending || !tokenName.trim()}
                onClick={() => createTokenMutation.mutate()}
                className="px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-xs font-semibold shadow"
              >
                {createTokenMutation.isPending ? 'Generating...' : 'Generate Token'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
