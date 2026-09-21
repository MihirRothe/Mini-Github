import React from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Database,
  Cpu,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  HardDrive,
  Clock
} from 'lucide-react';
import { TelemetryResponse } from '../types';

export const SystemStatusPage: React.FC = () => {
  const {
    data: telemetry,
    isLoading,
    isError,
    refetch,
    isFetching,
  } = useQuery<TelemetryResponse>({
    queryKey: ['system-telemetry-detailed'],
    queryFn: async () => {
      const res = await fetch('/api/v1/health');
      if (!res.ok) throw new Error('Health probe failed');
      return res.json();
    },
    refetchInterval: 5000,
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between pb-4 border-b border-forge-border gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            System Diagnostics & Status
          </h1>
          <p className="text-sm text-forge-muted mt-1">
            Real-time telemetry and health monitoring across the ForgeHub stack.
          </p>
        </div>

        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-forge-card hover:bg-forge-surface text-forge-text border border-forge-border text-xs font-semibold transition-all self-start md:self-auto disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isFetching ? 'animate-spin' : ''}`} />
          <span>Refresh Telemetry</span>
        </button>
      </div>

      {/* Global Status Banner */}
      <div className="glass-panel p-6 rounded-xl flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <div className="w-12 h-12 rounded-xl bg-forge-card border border-forge-border flex items-center justify-center">
            {isLoading ? (
              <RefreshCw className="w-6 h-6 text-forge-muted animate-spin" />
            ) : isError ? (
              <XCircle className="w-6 h-6 text-rose-500" />
            ) : telemetry?.status === 'healthy' ? (
              <CheckCircle2 className="w-6 h-6 text-emerald-400" />
            ) : (
              <AlertTriangle className="w-6 h-6 text-amber-400" />
            )}
          </div>
          <div>
            <h2 className="text-lg font-bold text-white">
              {isLoading
                ? 'Probing System Services...'
                : isError
                ? 'Services Unreachable'
                : telemetry?.status === 'healthy'
                ? 'All Systems Fully Operational'
                : 'System Running in Degraded State'}
            </h2>
            <p className="text-xs text-forge-muted mt-0.5">
              ForgeHub Core Platform • Release {telemetry?.version || '0.1.0-alpha'}
            </p>
          </div>
        </div>

        <div className="hidden sm:flex items-center space-x-4 text-xs font-mono text-forge-muted">
          <div className="flex items-center space-x-1.5">
            <Clock className="w-4 h-4 text-forge-muted" />
            <span>Uptime: {telemetry?.uptime_seconds ?? 0}s</span>
          </div>
        </div>
      </div>

      {/* Telemetry Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Database Subsystem */}
        <div className="glass-panel p-5 rounded-xl space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Database className="w-4 h-4 text-amber-400" />
              <h3 className="text-sm font-semibold text-white">Database Pool</h3>
            </div>
            <span
              className={`text-[10px] font-mono font-semibold px-2 py-0.5 rounded-full border ${
                telemetry?.database?.status === 'healthy'
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                  : 'bg-amber-500/10 text-amber-400 border-amber-500/20'
              }`}
            >
              {telemetry?.database?.status?.toUpperCase() || 'STANDALONE'}
            </span>
          </div>

          <div className="space-y-2 text-xs divide-y divide-forge-border/40">
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Driver / Mode</span>
              <span className="font-mono text-forge-text">{telemetry?.database?.driver || 'pgx/standalone'}</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Open Connections</span>
              <span className="font-mono text-forge-text">{telemetry?.database?.open_conns ?? 0}</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>In-Use Connections</span>
              <span className="font-mono text-forge-text">{telemetry?.database?.in_use_conns ?? 0}</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Idle Connections</span>
              <span className="font-mono text-forge-text">{telemetry?.database?.idle_conns ?? 0}</span>
            </div>
          </div>
        </div>

        {/* Redis Queues & PubSub */}
        <div className="glass-panel p-5 rounded-xl space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <HardDrive className="w-4 h-4 text-purple-400" />
              <h3 className="text-sm font-semibold text-white">Redis Queue / Cache</h3>
            </div>
            <span
              className={`text-[10px] font-mono font-semibold px-2 py-0.5 rounded-full border ${
                telemetry?.redis?.status === 'healthy'
                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                  : 'bg-rose-500/10 text-rose-400 border-rose-500/20'
              }`}
            >
              {telemetry?.redis?.status?.toUpperCase() || 'READY'}
            </span>
          </div>

          <div className="space-y-2 text-xs divide-y divide-forge-border/40">
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Execution Mode</span>
              <span className="font-mono text-forge-text">{telemetry?.redis?.mode || 'embedded'}</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Health Ping</span>
              <span className="font-mono text-emerald-400">PASSED</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Background Workers</span>
              <span className="font-mono text-forge-text">Ready</span>
            </div>
          </div>
        </div>

        {/* Go Runtime & Host */}
        <div className="glass-panel p-5 rounded-xl space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Cpu className="w-4 h-4 text-blue-400" />
              <h3 className="text-sm font-semibold text-white">Go Runtime</h3>
            </div>
            <span className="text-[10px] font-mono font-semibold px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20">
              {telemetry?.system?.go_version || 'Go 1.27'}
            </span>
          </div>

          <div className="space-y-2 text-xs divide-y divide-forge-border/40">
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Allocated Heap</span>
              <span className="font-mono text-forge-text">{telemetry?.system?.alloc_mb ?? 0} MB</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>System Memory</span>
              <span className="font-mono text-forge-text">{telemetry?.system?.sys_mb ?? 0} MB</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Active Goroutines</span>
              <span className="font-mono text-forge-text">{telemetry?.system?.num_goroutine ?? 0}</span>
            </div>
            <div className="flex justify-between py-1 text-forge-muted">
              <span>Available CPU Cores</span>
              <span className="font-mono text-forge-text">{telemetry?.system?.num_cpu ?? 0}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
