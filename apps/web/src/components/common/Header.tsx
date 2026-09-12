import React, { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import {
  GitBranch,
  Search,
  Bell,
  User as UserIcon,
  Settings,
  Key,
  LogOut,
  ChevronDown
} from 'lucide-react';
import { TelemetryResponse } from '../../types';

interface HeaderProps {
  onOpenCommandPalette: () => void;
}

export const Header: React.FC<HeaderProps> = ({ onOpenCommandPalette }) => {
  const { user, isAuthenticated, logout } = useAuth();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

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

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleLogout = async () => {
    setIsDropdownOpen(false);
    await logout();
    navigate('/login');
  };

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
                v0.2
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

        {/* Authenticated User Menu or Guest Buttons */}
        {isAuthenticated && user ? (
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setIsDropdownOpen(!isDropdownOpen)}
              className="flex items-center space-x-1.5 p-1 rounded-lg hover:bg-forge-card transition-colors"
            >
              <div className="w-7 h-7 rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-xs font-semibold text-white shadow overflow-hidden">
                {user.avatar_url ? (
                  <img src={user.avatar_url} alt={user.username} className="w-full h-full object-cover" />
                ) : (
                  user.username.slice(0, 2).toUpperCase()
                )}
              </div>
              <ChevronDown className="w-3.5 h-3.5 text-forge-muted" />
            </button>

            {isDropdownOpen && (
              <div className="absolute right-0 mt-2 w-56 bg-forge-surface border border-forge-border rounded-xl shadow-2xl py-1 text-xs z-50 animate-fade-in">
                <div className="px-4 py-2.5 border-b border-forge-border">
                  <div className="text-[11px] text-forge-muted">Signed in as</div>
                  <div className="font-semibold text-white truncate">@{user.username}</div>
                  {user.is_admin && (
                    <span className="inline-block mt-1 px-1.5 py-0.2 rounded bg-blue-500/10 text-blue-400 border border-blue-500/20 text-[10px] font-mono">
                      Administrator
                    </span>
                  )}
                </div>

                <div className="py-1">
                  <Link
                    to={`/${user.username}`}
                    onClick={() => setIsDropdownOpen(false)}
                    className="flex items-center space-x-2 px-4 py-2 hover:bg-forge-card text-forge-text transition-colors"
                  >
                    <UserIcon className="w-3.5 h-3.5 text-forge-muted" />
                    <span>Your Profile</span>
                  </Link>
                  <Link
                    to="/settings/profile"
                    onClick={() => setIsDropdownOpen(false)}
                    className="flex items-center space-x-2 px-4 py-2 hover:bg-forge-card text-forge-text transition-colors"
                  >
                    <Settings className="w-3.5 h-3.5 text-forge-muted" />
                    <span>Settings</span>
                  </Link>
                  <Link
                    to="/settings/tokens"
                    onClick={() => setIsDropdownOpen(false)}
                    className="flex items-center space-x-2 px-4 py-2 hover:bg-forge-card text-forge-text transition-colors"
                  >
                    <Key className="w-3.5 h-3.5 text-forge-muted" />
                    <span>Access Tokens</span>
                  </Link>
                </div>

                <div className="border-t border-forge-border pt-1">
                  <button
                    onClick={handleLogout}
                    className="w-full flex items-center space-x-2 px-4 py-2 text-rose-400 hover:bg-rose-500/10 transition-colors text-left"
                  >
                    <LogOut className="w-3.5 h-3.5" />
                    <span>Sign out</span>
                  </button>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="flex items-center space-x-2">
            <Link
              to="/login"
              className="px-2.5 py-1 text-xs font-semibold text-forge-muted hover:text-forge-text transition-colors"
            >
              Sign In
            </Link>
            <Link
              to="/register"
              className="px-3 py-1 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-md shadow-blue-600/20 transition-all"
            >
              Sign Up
            </Link>
          </div>
        )}
      </div>
    </header>
  );
};
