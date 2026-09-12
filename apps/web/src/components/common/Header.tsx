import React from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { GitBranch, Search, Bell } from 'lucide-react';
import { TelemetryResponse } from '../../types';

interface HeaderProps {
  onOpenCommandPalette: () => void;
}

export const Header: React.FC<HeaderProps> = ({ onOpenCommandPalette }) => {
  const { data: telemetry, isError } = useQuery<TelemetryResponse>({
    queryKey: ['system-health'],
    queryFn: async () => {
      const res = await fetch('/api/v1/health');
      if (!res.ok) throw new Error('Health check failed');
      return res.json();
    },
    refetchInterval: 10000,
    retry: 1,
  });

  const isOperational = !isError && telemetry?.status === 'healthy';
  const isDegraded = telemetry?.status === 'degraded';

  return (
    <header className="h-14 border-b border-forge-border bg-forge-surface/80 backdrop-blur-md sticky top-0 z-40 px-4 flex items-center justify-between">
      {/* Brand & Workspace */}
      <div className="flex items-center space-x-4">
        <Link to="/dashboard" className="flex items-center space-x-2.5 group">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center shadow-lg shadow-blue-500/20 group-hover:scale-105 transition-transform">
            <GitBranch className="w-5 h-5 text-white" />
          </div>
          <div className="flex flex-col">
            <span className="font-bold text-sm tracking-tight text-white flex items-center gap-1.5">
              ForgeHub
              <span className="text-[10px] uppercase font-mono px-1.5 py-0.2 bg-blue-500/10 text-blue-400 border border-blue-500/20 rounded">
                v0.1
              </span>
            </span>
          </div>
        </Link>
      </div>

      {/* Global Search & Command Trigger */}
      <div className="flex-1 max-w-md mx-6">
        <button
          onClick={onOpenCommandPalette}
          className="w-full flex items-center justify-between px-3 py-1.5 rounded-lg bg-forge-card border border-forge-border hover:border-forge-subtle text-forge-muted hover:text-forge-text text-xs transition-colors"
        >
          <div className="flex items-center space-x-2">
            <Search className="w-3.5 h-3.5" />
            <span>Search repositories, pull requests, issues...</span>
          </div>
          <kbd className="px-1.5 py-0.5 text-[10px] font-mono bg-forge-surface border border-forge-border rounded text-forge-muted">
            Ctrl K
          </kbd>
        </button>
      </div>

      {/* Right Controls: Telemetry, Notifications, Profile */}
      <div className="flex items-center space-x-3">
        {/* System Health Status Indicator */}
        <Link
          to="/status"
          className="flex items-center space-x-1.5 px-2.5 py-1 rounded-full bg-forge-card border border-forge-border hover:border-forge-subtle text-xs transition-colors"
          title={`API Status: ${telemetry?.status || (isError ? 'offline' : 'connecting')}`}
        >
          <span className="relative flex h-2 w-2">
            {isOperational && (
              <>
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
              </>
            )}
            {isDegraded && (
              <span className="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
            )}
            {(!isOperational && !isDegraded) && (
              <span className="relative inline-flex rounded-full h-2 w-2 bg-rose-500"></span>
            )}
          </span>
          <span className="text-[11px] font-medium text-forge-muted">
            {isOperational ? 'API Healthy' : isDegraded ? 'Degraded' : 'Offline'}
          </span>
        </Link>

        {/* Notifications */}
        <button
          className="p-1.5 text-forge-muted hover:text-forge-text hover:bg-forge-card rounded-lg transition-colors relative"
          title="Notifications"
        >
          <Bell className="w-4 h-4" />
        </button>

        {/* User avatar placeholder */}
        <div className="w-7 h-7 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-xs font-semibold text-white shadow">
          FH
        </div>
      </div>
    </header>
  );
};
