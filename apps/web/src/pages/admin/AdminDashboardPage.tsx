import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import {
  Shield,
  Users,
  GitBranch,
  Building2,
  PlayCircle,
  FileText,
  AlertTriangle,
  CheckCircle,
  Clock,
  Cpu,
  HardDrive,
  RefreshCw,
  Search,
  UserCheck,
  UserX,
  ShieldAlert,
  FolderGit2,
  Terminal,
  Activity,
  Loader2,
  Radio
} from 'lucide-react';

interface AdminStats {
  total_users: number;
  active_users: number;
  suspended_users: number;
  admin_users: number;
  total_organizations: number;
  total_repositories: number;
  public_repositories: number;
  private_repositories: number;
  total_issues: number;
  open_issues: number;
  closed_issues: number;
  total_pull_requests: number;
  open_pull_requests: number;
  merged_pull_requests: number;
  total_pipelines: number;
  queued_pipelines: number;
  running_pipelines: number;
  total_webhooks: number;
  goroutines_count: number;
  allocated_mem_bytes: number;
  sys_mem_bytes: number;
  gc_cycles: number;
  uptime_seconds: number;
  timestamp: string;
}

interface AuditLogEntry {
  id: string;
  actor_id?: string;
  actor_username?: string;
  action: string;
  target_type: string;
  target_id: string;
  ip_address: string;
  user_agent: string;
  metadata: any;
  created_at: string;
}

interface AdminUserItem {
  id: string;
  username: string;
  email: string;
  display_name: string;
  avatar_url: string;
  is_admin: boolean;
  is_suspended: boolean;
  repos_count: number;
  created_at: string;
  updated_at: string;
}

interface AdminOrgItem {
  id: string;
  name: string;
  slug: string;
  description: string;
  member_count: number;
  repos_count: number;
  created_at: string;
}

interface AdminRepoItem {
  id: string;
  name: string;
  slug: string;
  owner_name: string;
  owner_type: string;
  visibility: string;
  default_branch: string;
  is_archived: boolean;
  created_at: string;
}

export const AdminDashboardPage: React.FC = () => {
  const { user, isAuthenticated, isLoading: authLoading } = useAuth();
  const [activeTab, setActiveTab] = useState<'overview' | 'users' | 'audit' | 'repos' | 'orgs'>('overview');

  const [stats, setStats] = useState<AdminStats | null>(null);
  const [statsLoading, setStatsLoading] = useState(true);

  const [usersList, setUsersList] = useState<AdminUserItem[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [userQuery, setUserQuery] = useState('');

  const [auditLogs, setAuditLogs] = useState<AuditLogEntry[]>([]);
  const [auditLoading, setAuditLoading] = useState(false);

  const [orgsList, setOrgsList] = useState<AdminOrgItem[]>([]);
  const [orgsLoading, setOrgsLoading] = useState(false);

  const [reposList, setReposList] = useState<AdminRepoItem[]>([]);
  const [reposLoading, setReposLoading] = useState(false);

  const [actionSuccess, setActionSuccess] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const fetchStats = async () => {
    setStatsLoading(true);
    try {
      const res = await fetch('/api/v1/admin/stats', { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setStats(data.stats);
      }
    } catch (err) {
      // Ignored
    } finally {
      setStatsLoading(false);
    }
  };

  const fetchUsers = async () => {
    setUsersLoading(true);
    try {
      const url = userQuery
        ? `/api/v1/admin/users?q=${encodeURIComponent(userQuery)}`
        : '/api/v1/admin/users';
      const res = await fetch(url, { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setUsersList(data.users || []);
      }
    } catch (err) {
      // Ignored
    } finally {
      setUsersLoading(false);
    }
  };

  const fetchAuditLogs = async () => {
    setAuditLoading(true);
    try {
      const res = await fetch('/api/v1/admin/audit-logs?limit=50', { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setAuditLogs(data.audit_logs || []);
      }
    } catch (err) {
      // Ignored
    } finally {
      setAuditLoading(false);
    }
  };

  const fetchOrgs = async () => {
    setOrgsLoading(true);
    try {
      const res = await fetch('/api/v1/admin/orgs', { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setOrgsList(data.organizations || []);
      }
    } catch (err) {
      // Ignored
    } finally {
      setOrgsLoading(false);
    }
  };

  const fetchRepos = async () => {
    setReposLoading(true);
    try {
      const res = await fetch('/api/v1/admin/repos', { credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setReposList(data.repositories || []);
      }
    } catch (err) {
      // Ignored
    } finally {
      setReposLoading(false);
    }
  };

  useEffect(() => {
    if (user?.is_admin) {
      fetchStats();
    }
  }, [user]);

  useEffect(() => {
    if (!user?.is_admin) return;
    if (activeTab === 'users') fetchUsers();
    if (activeTab === 'audit') fetchAuditLogs();
    if (activeTab === 'orgs') fetchOrgs();
    if (activeTab === 'repos') fetchRepos();
  }, [activeTab, user]);

  const handleUpdateUserStatus = async (
    targetUserId: string,
    updates: { is_admin?: boolean; is_suspended?: boolean }
  ) => {
    setActionSuccess(null);
    setActionError(null);
    try {
      const res = await fetch(`/api/v1/admin/users/${targetUserId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(updates),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to update user status');
      }

      setActionSuccess(`User status successfully updated for @${data.user.username}`);
      setUsersList((prev) =>
        prev.map((u) => (u.id === targetUserId ? { ...u, ...data.user } : u))
      );
      // Refresh stats in background
      fetchStats();
    } catch (err: any) {
      setActionError(err.message || 'Operation failed');
    }
  };

  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 MB';
    const mb = bytes / (1024 * 1024);
    return mb.toFixed(1) + ' MB';
  };

  const formatUptime = (seconds: number) => {
    if (!seconds) return '0m';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    if (hrs > 0) return `${hrs}h ${mins}m`;
    return `${mins}m`;
  };

  if (authLoading) {
    return (
      <div className="py-24 text-center">
        <Loader2 className="w-8 h-8 text-blue-500 animate-spin mx-auto mb-3" />
        <p className="text-xs text-forge-muted">Verifying administrator credentials...</p>
      </div>
    );
  }

  if (!isAuthenticated || !user?.is_admin) {
    return (
      <div className="max-w-xl mx-auto py-20 px-4 text-center space-y-4">
        <div className="w-16 h-16 rounded-2xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center text-rose-400 mx-auto">
          <ShieldAlert className="w-8 h-8" />
        </div>
        <h2 className="text-xl font-bold text-white">Administrator Access Required</h2>
        <p className="text-xs text-forge-muted leading-relaxed">
          This dashboard contains sensitive platform telemetry, audit event streams, and user authority management. Your account does not have site administrator privileges.
        </p>
        <Link to="/dashboard" className="btn-secondary inline-block text-xs">
          Return to Dashboard
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-8 pb-12">
      {/* Header Banner */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-6 border-b border-forge-border">
        <div className="flex items-center space-x-3.5">
          <div className="w-11 h-11 rounded-2xl bg-gradient-to-tr from-rose-600 to-purple-600 flex items-center justify-center shadow-lg shadow-rose-600/20 text-white">
            <Shield className="w-6 h-6" />
          </div>
          <div>
            <div className="flex items-center gap-2.5">
              <h1 className="text-2xl font-bold text-white tracking-tight">Admin & Observability</h1>
              <span className="px-2 py-0.5 rounded-full text-[10px] font-mono font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">
                Superuser
              </span>
            </div>
            <p className="text-xs text-forge-muted">
              Platform telemetry, system metrics, user governance, and security audit log
            </p>
          </div>
        </div>

        <div className="flex items-center space-x-2">
          <button
            onClick={() => {
              fetchStats();
              if (activeTab === 'users') fetchUsers();
              if (activeTab === 'audit') fetchAuditLogs();
              if (activeTab === 'orgs') fetchOrgs();
              if (activeTab === 'repos') fetchRepos();
            }}
            className="btn-secondary text-xs flex items-center gap-1.5"
            title="Refresh statistics"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${statsLoading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
        </div>
      </div>

      {/* Notifications */}
      {actionSuccess && (
        <div className="p-3.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center justify-between animate-fade-in">
          <div className="flex items-center gap-2">
            <CheckCircle className="w-4 h-4" />
            <span>{actionSuccess}</span>
          </div>
          <button onClick={() => setActionSuccess(null)} className="text-forge-muted hover:text-white text-xs">
            ✕
          </button>
        </div>
      )}

      {actionError && (
        <div className="p-3.5 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs flex items-center justify-between animate-fade-in">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-4 h-4" />
            <span>{actionError}</span>
          </div>
          <button onClick={() => setActionError(null)} className="text-forge-muted hover:text-white text-xs">
            ✕
          </button>
        </div>
      )}

      {/* Primary KPI Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        <div className="card p-3.5 space-y-1">
          <div className="flex items-center justify-between text-forge-muted text-[11px]">
            <span>Total Users</span>
            <Users className="w-3.5 h-3.5 text-blue-400" />
          </div>
          <div className="text-xl font-bold text-white">{stats?.total_users ?? 0}</div>
          <div className="text-[10px] text-forge-muted">
            <span className="text-emerald-400 font-semibold">{stats?.active_users ?? 0}</span> active
          </div>
        </div>

        <div className="card p-3.5 space-y-1">
          <div className="flex items-center justify-between text-forge-muted text-[11px]">
            <span>Repositories</span>
            <FolderGit2 className="w-3.5 h-3.5 text-purple-400" />
          </div>
          <div className="text-xl font-bold text-white">{stats?.total_repositories ?? 0}</div>
          <div className="text-[10px] text-forge-muted">
            {stats?.public_repositories ?? 0} public · {stats?.private_repositories ?? 0} private
          </div>
        </div>

        <div className="card p-3.5 space-y-1">
          <div className="flex items-center justify-between text-forge-muted text-[11px]">
            <span>Organizations</span>
            <Building2 className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <div className="text-xl font-bold text-white">{stats?.total_organizations ?? 0}</div>
          <div className="text-[10px] text-forge-muted">Active teams & workspaces</div>
        </div>

        <div className="card p-3.5 space-y-1">
          <div className="flex items-center justify-between text-forge-muted text-[11px]">
            <span>CI Pipelines</span>
            <PlayCircle className="w-3.5 h-3.5 text-emerald-400" />
          </div>
          <div className="text-xl font-bold text-white">{stats?.total_pipelines ?? 0}</div>
          <div className="text-[10px] text-forge-muted">
            {stats?.running_pipelines ?? 0} running · {stats?.queued_pipelines ?? 0} queued
          </div>
        </div>

        <div className="card p-3.5 space-y-1">
          <div className="flex items-center justify-between text-forge-muted text-[11px]">
            <span>Pull Requests</span>
            <GitBranch className="w-3.5 h-3.5 text-indigo-400" />
          </div>
          <div className="text-xl font-bold text-white">{stats?.total_pull_requests ?? 0}</div>
          <div className="text-[10px] text-forge-muted">
            {stats?.open_pull_requests ?? 0} open · {stats?.merged_pull_requests ?? 0} merged
          </div>
        </div>

        <div className="card p-3.5 space-y-1">
          <div className="flex items-center justify-between text-forge-muted text-[11px]">
            <span>Webhooks</span>
            <Radio className="w-3.5 h-3.5 text-rose-400" />
          </div>
          <div className="text-xl font-bold text-white">{stats?.total_webhooks ?? 0}</div>
          <div className="text-[10px] text-forge-muted">Active event dispatchers</div>
        </div>
      </div>

      {/* Navigation Tabs */}
      <div className="flex border-b border-forge-border bg-forge-card/40 px-2 rounded-xl overflow-x-auto text-xs no-scrollbar">
        <button
          onClick={() => setActiveTab('overview')}
          className={`py-3 px-4 font-semibold border-b-2 transition-colors flex items-center gap-2 whitespace-nowrap ${
            activeTab === 'overview'
              ? 'border-blue-500 text-blue-400'
              : 'border-transparent text-forge-muted hover:text-white'
          }`}
        >
          <Activity className="w-3.5 h-3.5" />
          <span>Overview & Health</span>
        </button>

        <button
          onClick={() => setActiveTab('users')}
          className={`py-3 px-4 font-semibold border-b-2 transition-colors flex items-center gap-2 whitespace-nowrap ${
            activeTab === 'users'
              ? 'border-blue-500 text-blue-400'
              : 'border-transparent text-forge-muted hover:text-white'
          }`}
        >
          <Users className="w-3.5 h-3.5" />
          <span>User Management</span>
        </button>

        <button
          onClick={() => setActiveTab('audit')}
          className={`py-3 px-4 font-semibold border-b-2 transition-colors flex items-center gap-2 whitespace-nowrap ${
            activeTab === 'audit'
              ? 'border-blue-500 text-blue-400'
              : 'border-transparent text-forge-muted hover:text-white'
          }`}
        >
          <FileText className="w-3.5 h-3.5" />
          <span>Audit Log Stream</span>
        </button>

        <button
          onClick={() => setActiveTab('repos')}
          className={`py-3 px-4 font-semibold border-b-2 transition-colors flex items-center gap-2 whitespace-nowrap ${
            activeTab === 'repos'
              ? 'border-blue-500 text-blue-400'
              : 'border-transparent text-forge-muted hover:text-white'
          }`}
        >
          <FolderGit2 className="w-3.5 h-3.5" />
          <span>Repositories</span>
        </button>

        <button
          onClick={() => setActiveTab('orgs')}
          className={`py-3 px-4 font-semibold border-b-2 transition-colors flex items-center gap-2 whitespace-nowrap ${
            activeTab === 'orgs'
              ? 'border-blue-500 text-blue-400'
              : 'border-transparent text-forge-muted hover:text-white'
          }`}
        >
          <Building2 className="w-3.5 h-3.5" />
          <span>Organizations</span>
        </button>
      </div>

      {/* Tab 1: Overview & Runtime Telemetry */}
      {activeTab === 'overview' && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="card p-4 space-y-2">
              <div className="flex items-center justify-between text-xs text-forge-muted">
                <span className="font-semibold text-white flex items-center gap-1.5">
                  <Cpu className="w-4 h-4 text-purple-400" />
                  Go Goroutines
                </span>
                <span className="text-emerald-400 font-mono">Normal</span>
              </div>
              <div className="text-2xl font-mono font-bold text-white">
                {stats?.goroutines_count ?? 0}
              </div>
              <p className="text-[11px] text-forge-muted">
                Active lightweight runtime threads handling HTTP requests and CI workers
              </p>
            </div>

            <div className="card p-4 space-y-2">
              <div className="flex items-center justify-between text-xs text-forge-muted">
                <span className="font-semibold text-white flex items-center gap-1.5">
                  <HardDrive className="w-4 h-4 text-blue-400" />
                  Allocated Heap
                </span>
                <span className="text-forge-muted font-mono">{formatBytes(stats?.sys_mem_bytes || 0)} Sys</span>
              </div>
              <div className="text-2xl font-mono font-bold text-white">
                {formatBytes(stats?.allocated_mem_bytes || 0)}
              </div>
              <p className="text-[11px] text-forge-muted">
                Currently active memory footprint across all backend modules
              </p>
            </div>

            <div className="card p-4 space-y-2">
              <div className="flex items-center justify-between text-xs text-forge-muted">
                <span className="font-semibold text-white flex items-center gap-1.5">
                  <Activity className="w-4 h-4 text-emerald-400" />
                  GC Cycles
                </span>
                <span className="text-emerald-400 font-mono">Healthy</span>
              </div>
              <div className="text-2xl font-mono font-bold text-white">
                {stats?.gc_cycles ?? 0}
              </div>
              <p className="text-[11px] text-forge-muted">
                Completed garbage collection cycles since application launch
              </p>
            </div>

            <div className="card p-4 space-y-2">
              <div className="flex items-center justify-between text-xs text-forge-muted">
                <span className="font-semibold text-white flex items-center gap-1.5">
                  <Clock className="w-4 h-4 text-amber-400" />
                  System Uptime
                </span>
                <span className="text-emerald-400 font-mono">Online</span>
              </div>
              <div className="text-2xl font-mono font-bold text-white">
                {formatUptime(stats?.uptime_seconds || 0)}
              </div>
              <p className="text-[11px] text-forge-muted">
                Continuous daemon uptime without interruptions or fatal restarts
              </p>
            </div>
          </div>

          {/* Quick Info Grid */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Security Audit Snapshot */}
            <div className="card p-5 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-white flex items-center gap-2">
                  <Shield className="w-4 h-4 text-rose-400" />
                  Security Governance
                </h3>
                <span className="text-xs text-forge-muted">{stats?.admin_users ?? 0} Superusers</span>
              </div>
              <div className="space-y-2 text-xs">
                <div className="flex justify-between py-2 border-b border-forge-border">
                  <span className="text-forge-muted">Active Administrators</span>
                  <span className="font-bold text-white font-mono">{stats?.admin_users ?? 0}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-forge-border">
                  <span className="text-forge-muted">Suspended Accounts</span>
                  <span className="font-bold text-rose-400 font-mono">{stats?.suspended_users ?? 0}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-forge-border">
                  <span className="text-forge-muted">Session Isolation</span>
                  <span className="font-mono text-emerald-400">Strict HttpOnly & Lax</span>
                </div>
                <div className="flex justify-between py-2">
                  <span className="text-forge-muted">AI Injection Shield</span>
                  <span className="font-mono text-purple-400">Active (Delimiter Boundaries)</span>
                </div>
              </div>
            </div>

            {/* Storage & Engine Snapshot */}
            <div className="card p-5 space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-sm font-bold text-white flex items-center gap-2">
                  <Terminal className="w-4 h-4 text-purple-400" />
                  Storage & Smart HTTP Engine
                </h3>
                <span className="text-xs text-forge-muted">{stats?.total_repositories ?? 0} Repos</span>
              </div>
              <div className="space-y-2 text-xs">
                <div className="flex justify-between py-2 border-b border-forge-border">
                  <span className="text-forge-muted">Bare Git Repositories</span>
                  <span className="font-bold text-white font-mono">{stats?.total_repositories ?? 0}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-forge-border">
                  <span className="text-forge-muted">Issues / Discussions</span>
                  <span className="font-bold text-white font-mono">{stats?.total_issues ?? 0}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-forge-border">
                  <span className="text-forge-muted">Pull Request Reviews</span>
                  <span className="font-bold text-white font-mono">{stats?.total_pull_requests ?? 0}</span>
                </div>
                <div className="flex justify-between py-2">
                  <span className="text-forge-muted">Smart HTTP Clone/Push</span>
                  <span className="font-mono text-emerald-400">Enabled</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Tab 2: User Management */}
      {activeTab === 'users' && (
        <div className="card overflow-hidden space-y-4 p-5">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-white">Registered Users</h3>
              <p className="text-xs text-forge-muted">
                Promote site administrators, review permissions, or suspend compromised accounts
              </p>
            </div>

            <form
              onSubmit={(e) => {
                e.preventDefault();
                fetchUsers();
              }}
              className="relative w-full sm:w-64"
            >
              <Search className="w-3.5 h-3.5 text-forge-muted absolute left-3 top-2.5" />
              <input
                type="text"
                value={userQuery}
                onChange={(e) => setUserQuery(e.target.value)}
                placeholder="Search username or email..."
                className="input pl-8 py-1.5 text-xs w-full"
              />
            </form>
          </div>

          {usersLoading ? (
            <div className="py-12 text-center">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500 mx-auto mb-2" />
              <span className="text-xs text-forge-muted">Loading user database...</span>
            </div>
          ) : usersList.length === 0 ? (
            <div className="p-8 text-center text-xs text-forge-muted">No users found matching query.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead>
                  <tr className="border-b border-forge-border text-forge-muted uppercase tracking-wider text-[10px]">
                    <th className="py-2.5 px-3">User</th>
                    <th className="py-2.5 px-3">Role</th>
                    <th className="py-2.5 px-3">Status</th>
                    <th className="py-2.5 px-3">Repositories</th>
                    <th className="py-2.5 px-3">Registered</th>
                    <th className="py-2.5 px-3 text-right">Actions</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-forge-border">
                  {usersList.map((u) => (
                    <tr key={u.id} className="hover:bg-forge-card/40 transition-colors">
                      <td className="py-3 px-3">
                        <div className="flex items-center space-x-2.5">
                          <div className="w-7 h-7 rounded-lg bg-forge-bg border border-forge-border flex items-center justify-center font-bold text-forge-text text-xs overflow-hidden">
                            {u.avatar_url ? (
                              <img src={u.avatar_url} alt={u.username} className="w-full h-full object-cover" />
                            ) : (
                              u.username.slice(0, 2).toUpperCase()
                            )}
                          </div>
                          <div>
                            <div className="font-semibold text-white">@{u.username}</div>
                            <div className="text-[11px] text-forge-muted">{u.email}</div>
                          </div>
                        </div>
                      </td>
                      <td className="py-3 px-3">
                        {u.is_admin ? (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-mono font-semibold bg-purple-500/15 text-purple-300 border border-purple-500/20">
                            Administrator
                          </span>
                        ) : (
                          <span className="text-forge-muted text-xs">Standard</span>
                        )}
                      </td>
                      <td className="py-3 px-3">
                        {u.is_suspended ? (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-rose-500/15 text-rose-400 border border-rose-500/20">
                            Suspended
                          </span>
                        ) : (
                          <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-500/15 text-emerald-400 border border-emerald-500/20">
                            Active
                          </span>
                        )}
                      </td>
                      <td className="py-3 px-3 font-mono">{u.repos_count}</td>
                      <td className="py-3 px-3 text-forge-muted">
                        {new Date(u.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-3 px-3 text-right">
                        <div className="flex items-center justify-end space-x-1.5">
                          {/* Toggle Admin */}
                          {user.id !== u.id && (
                            <button
                              onClick={() => handleUpdateUserStatus(u.id, { is_admin: !u.is_admin })}
                              className="p-1.5 rounded-lg hover:bg-forge-bg text-forge-muted hover:text-white transition-colors"
                              title={u.is_admin ? 'Demote from Admin' : 'Promote to Admin'}
                            >
                              <Shield className={`w-3.5 h-3.5 ${u.is_admin ? 'text-purple-400' : ''}`} />
                            </button>
                          )}

                          {/* Toggle Suspension */}
                          {user.id !== u.id && (
                            <button
                              onClick={() => handleUpdateUserStatus(u.id, { is_suspended: !u.is_suspended })}
                              className={`p-1.5 rounded-lg hover:bg-forge-bg transition-colors ${
                                u.is_suspended ? 'text-rose-400' : 'text-forge-muted hover:text-rose-400'
                              }`}
                              title={u.is_suspended ? 'Unsuspend Account' : 'Suspend Account'}
                            >
                              {u.is_suspended ? <UserCheck className="w-3.5 h-3.5 text-emerald-400" /> : <UserX className="w-3.5 h-3.5" />}
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Tab 3: Audit Log Stream */}
      {activeTab === 'audit' && (
        <div className="card overflow-hidden space-y-4 p-5">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-white">Audit Log Stream</h3>
              <p className="text-xs text-forge-muted">
                Cryptographically tracked administrative, membership, and security transactions
              </p>
            </div>
            <button onClick={fetchAuditLogs} className="btn-secondary text-xs flex items-center gap-1">
              <RefreshCw className={`w-3 h-3 ${auditLoading ? 'animate-spin' : ''}`} />
              <span>Refresh</span>
            </button>
          </div>

          {auditLoading ? (
            <div className="py-12 text-center">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500 mx-auto mb-2" />
              <span className="text-xs text-forge-muted">Fetching audit timeline...</span>
            </div>
          ) : auditLogs.length === 0 ? (
            <div className="p-8 text-center text-xs text-forge-muted">
              No audit events recorded yet. Administrative actions will populate here automatically.
            </div>
          ) : (
            <div className="divide-y divide-forge-border text-xs">
              {auditLogs.map((log) => (
                <div key={log.id} className="py-3 px-2 hover:bg-forge-card/30 transition-colors flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-mono font-semibold uppercase ${
                          log.action.startsWith('admin.')
                            ? 'bg-purple-500/15 text-purple-300 border border-purple-500/20'
                            : log.action.startsWith('repo.')
                            ? 'bg-blue-500/15 text-blue-300 border border-blue-500/20'
                            : 'bg-emerald-500/15 text-emerald-300 border border-emerald-500/20'
                        }`}
                      >
                        {log.action}
                      </span>
                      <span className="font-semibold text-white">
                        {log.actor_username ? `@${log.actor_username}` : 'System'}
                      </span>
                      <span className="text-forge-muted">targeted</span>
                      <span className="font-mono text-purple-300 bg-forge-bg px-1.5 py-0.2 rounded border border-forge-border">
                        {log.target_type}:{log.target_id}
                      </span>
                    </div>

                    {log.metadata && Object.keys(log.metadata).length > 0 && (
                      <div className="text-[11px] font-mono text-forge-muted bg-forge-bg/60 p-2 rounded border border-forge-border/40 overflow-x-auto">
                        {JSON.stringify(log.metadata)}
                      </div>
                    )}
                  </div>

                  <div className="text-right text-[11px] text-forge-muted shrink-0">
                    <div>{new Date(log.created_at).toLocaleString()}</div>
                    {log.ip_address && <div className="font-mono text-[10px]">{log.ip_address}</div>}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Tab 4: Repositories Inventory */}
      {activeTab === 'repos' && (
        <div className="card overflow-hidden space-y-4 p-5">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-white">Repositories Inventory</h3>
              <p className="text-xs text-forge-muted">
                System-wide Git repository hosting directory and storage allocation
              </p>
            </div>
          </div>

          {reposLoading ? (
            <div className="py-12 text-center">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500 mx-auto mb-2" />
              <span className="text-xs text-forge-muted">Enumerating repositories...</span>
            </div>
          ) : reposList.length === 0 ? (
            <div className="p-8 text-center text-xs text-forge-muted">No repositories found.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead>
                  <tr className="border-b border-forge-border text-forge-muted uppercase tracking-wider text-[10px]">
                    <th className="py-2.5 px-3">Repository</th>
                    <th className="py-2.5 px-3">Owner</th>
                    <th className="py-2.5 px-3">Visibility</th>
                    <th className="py-2.5 px-3">Branch</th>
                    <th className="py-2.5 px-3">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-forge-border">
                  {reposList.map((r) => (
                    <tr key={r.id} className="hover:bg-forge-card/40 transition-colors">
                      <td className="py-3 px-3">
                        <Link to={`/${r.owner_name}/${r.slug}`} className="font-semibold text-white hover:text-blue-400 flex items-center gap-1.5">
                          <FolderGit2 className="w-3.5 h-3.5 text-purple-400" />
                          <span>{r.slug}</span>
                        </Link>
                      </td>
                      <td className="py-3 px-3 font-mono text-forge-muted">
                        @{r.owner_name} ({r.owner_type})
                      </td>
                      <td className="py-3 px-3">
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-mono uppercase bg-forge-bg text-forge-muted border border-forge-border">
                          {r.visibility}
                        </span>
                      </td>
                      <td className="py-3 px-3 font-mono text-xs text-forge-muted">{r.default_branch}</td>
                      <td className="py-3 px-3 text-forge-muted">{new Date(r.created_at).toLocaleDateString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Tab 5: Organizations Inventory */}
      {activeTab === 'orgs' && (
        <div className="card overflow-hidden space-y-4 p-5">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-white">Organizations Inventory</h3>
              <p className="text-xs text-forge-muted">
                Platform organizations and workspace structures
              </p>
            </div>
          </div>

          {orgsLoading ? (
            <div className="py-12 text-center">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500 mx-auto mb-2" />
              <span className="text-xs text-forge-muted">Enumerating organizations...</span>
            </div>
          ) : orgsList.length === 0 ? (
            <div className="p-8 text-center text-xs text-forge-muted">No organizations registered yet.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead>
                  <tr className="border-b border-forge-border text-forge-muted uppercase tracking-wider text-[10px]">
                    <th className="py-2.5 px-3">Organization</th>
                    <th className="py-2.5 px-3">Slug</th>
                    <th className="py-2.5 px-3">Members</th>
                    <th className="py-2.5 px-3">Repositories</th>
                    <th className="py-2.5 px-3">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-forge-border">
                  {orgsList.map((o) => (
                    <tr key={o.id} className="hover:bg-forge-card/40 transition-colors">
                      <td className="py-3 px-3">
                        <Link to={`/orgs/${o.slug}`} className="font-semibold text-white hover:text-blue-400 flex items-center gap-1.5">
                          <Building2 className="w-3.5 h-3.5 text-amber-400" />
                          <span>{o.name}</span>
                        </Link>
                      </td>
                      <td className="py-3 px-3 font-mono text-forge-muted">{o.slug}</td>
                      <td className="py-3 px-3 font-mono">{o.member_count}</td>
                      <td className="py-3 px-3 font-mono">{o.repos_count}</td>
                      <td className="py-3 px-3 text-forge-muted">{new Date(o.created_at).toLocaleDateString()}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
