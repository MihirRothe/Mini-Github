import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  FolderGit2,
  GitPullRequest,
  CheckCircle2,
  PlayCircle,
  Plus,
  Building2,
  Activity,
  ArrowRight,
  Server,
  Database,
  Cpu
} from 'lucide-react';
import { TelemetryResponse } from '../types';

export const DashboardPage: React.FC = () => {
  const { data: telemetry, isLoading, isError } = useQuery<TelemetryResponse>({
    queryKey: ['system-health'],
    queryFn: async () => {
      const res = await fetch('/api/v1/health');
      if (!res.ok) throw new Error('Failed to fetch system telemetry');
      return res.json();
    },
    refetchInterval: 10000,
  });

  const stats = [
    { label: 'Repositories', count: '0', icon: FolderGit2, desc: 'Hosted Git projects' },
    { label: 'Pull Requests', count: '0', icon: GitPullRequest, desc: 'Awaiting code review' },
    { label: 'Assigned Issues', count: '0', icon: CheckCircle2, desc: 'Assigned to your account' },
    { label: 'Active Workflows', count: '0', icon: PlayCircle, desc: 'CI/CD runner jobs' },
  ];

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between pb-4 border-b border-forge-border gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            Developer Dashboard
          </h1>
          <p className="text-sm text-forge-muted mt-1">
            Welcome to ForgeHub. Manage repositories, collaborate on code, and monitor CI workflows.
          </p>
        </div>

        <div className="flex items-center space-x-2.5">
          <Link
            to="/explore"
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/20 transition-all"
          >
            <Plus className="w-4 h-4" />
            <span>New Repository</span>
          </Link>
          <button
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-forge-card hover:bg-forge-surface text-forge-text border border-forge-border text-xs font-semibold transition-all"
          >
            <Building2 className="w-4 h-4 text-forge-muted" />
            <span>New Organization</span>
          </button>
        </div>
      </div>

      {/* Metric Counters */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {stats.map((stat, idx) => {
          const Icon = stat.icon;
          return (
            <div key={idx} className="glass-card p-4 rounded-xl space-y-2">
              <div className="flex items-center justify-between text-forge-muted">
                <span className="text-xs font-medium uppercase tracking-wider">{stat.label}</span>
                <Icon className="w-4 h-4 text-forge-muted" />
              </div>
              <div className="text-2xl font-bold text-white tracking-tight">{stat.count}</div>
              <div className="text-[11px] text-forge-muted">{stat.desc}</div>
            </div>
          );
        })}
      </div>

      {/* Main Split: Repositories & Real-time Telemetry */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left 2 Cols: Repositories & Activity */}
        <div className="lg:col-span-2 space-y-6">
          {/* Repositories Card */}
          <div className="glass-panel rounded-xl overflow-hidden">
            <div className="px-5 py-4 border-b border-forge-border flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <FolderGit2 className="w-4 h-4 text-blue-400" />
                <h2 className="text-sm font-semibold text-white">Your Repositories</h2>
              </div>
              <Link to="/explore" className="text-xs text-blue-400 hover:underline flex items-center space-x-1">
                <span>View all</span>
                <ArrowRight className="w-3 h-3" />
              </Link>
            </div>

            <div className="p-8 text-center space-y-3">
              <div className="w-12 h-12 rounded-xl bg-forge-card border border-forge-border flex items-center justify-center mx-auto text-forge-muted">
                <FolderGit2 className="w-6 h-6" />
              </div>
              <h3 className="text-sm font-semibold text-white">No repositories yet</h3>
              <p className="text-xs text-forge-muted max-w-sm mx-auto">
                Get started by creating a new repository or importing an existing codebase to collaborate with your team.
              </p>
              <div className="pt-2">
                <Link
                  to="/explore"
                  className="inline-flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-forge-card hover:bg-forge-subtle border border-forge-border text-white text-xs font-medium transition-colors"
                >
                  <Plus className="w-3.5 h-3.5 text-blue-400" />
                  <span>Create your first repository</span>
                </Link>
              </div>
            </div>
          </div>

          {/* Recent Activity */}
          <div className="glass-panel rounded-xl overflow-hidden">
            <div className="px-5 py-4 border-b border-forge-border flex items-center justify-between">
              <h2 className="text-sm font-semibold text-white">Recent Activity</h2>
            </div>
            <div className="p-8 text-center text-xs text-forge-muted">
              No recent audit events or commit activities recorded yet.
            </div>
          </div>
        </div>

        {/* Right Col: Live Cluster Telemetry */}
        <div className="space-y-6">
          <div className="glass-panel rounded-xl overflow-hidden">
            <div className="px-5 py-4 border-b border-forge-border flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Activity className="w-4 h-4 text-emerald-400" />
                <h2 className="text-sm font-semibold text-white">System Diagnostics</h2>
              </div>
              <Link to="/status" className="text-xs text-blue-400 hover:underline">
                Details
              </Link>
            </div>

            <div className="p-5 space-y-4">
              {isLoading ? (
                <div className="py-6 text-center text-xs text-forge-muted">Checking backend status...</div>
              ) : isError ? (
                <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-lg text-xs text-rose-300">
                  Backend API is unreachable at <code className="font-mono">http://localhost:8080</code>.
                </div>
              ) : (
                <div className="space-y-3">
                  <div className="flex items-center justify-between text-xs py-1.5 border-b border-forge-border/40">
                    <span className="flex items-center gap-1.5 text-forge-muted">
                      <Server className="w-3.5 h-3.5 text-blue-400" /> API Gateway
                    </span>
                    <span className="font-mono text-emerald-400 text-[11px] font-semibold">
                      {telemetry?.status.toUpperCase()}
                    </span>
                  </div>

                  <div className="flex items-center justify-between text-xs py-1.5 border-b border-forge-border/40">
                    <span className="flex items-center gap-1.5 text-forge-muted">
                      <Database className="w-3.5 h-3.5 text-amber-400" /> Database Driver
                    </span>
                    <span className="font-mono text-forge-text text-[11px]">
                      {telemetry?.database.driver} ({telemetry?.database.status})
                    </span>
                  </div>

                  <div className="flex items-center justify-between text-xs py-1.5 border-b border-forge-border/40">
                    <span className="flex items-center gap-1.5 text-forge-muted">
                      <Cpu className="w-3.5 h-3.5 text-purple-400" /> Redis Mode
                    </span>
                    <span className="font-mono text-forge-text text-[11px]">
                      {telemetry?.redis.mode}
                    </span>
                  </div>

                  <div className="flex items-center justify-between text-xs py-1.5 border-b border-forge-border/40">
                    <span className="text-forge-muted">Uptime</span>
                    <span className="font-mono text-forge-text text-[11px]">
                      {telemetry?.uptime_seconds}s
                    </span>
                  </div>

                  <div className="flex items-center justify-between text-xs py-1.5">
                    <span className="text-forge-muted">Go Runtime</span>
                    <span className="font-mono text-forge-text text-[11px]">
                      {telemetry?.system.go_version}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
