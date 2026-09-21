import React from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Compass,
  FolderGit2,
  GitPullRequest,
  CheckSquare,
  PlaySquare,
  Building2,
  Activity
} from 'lucide-react';

export const Sidebar: React.FC = () => {
  const navItems = [
    { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { to: '/explore', label: 'Explore Repos', icon: Compass },
    { to: '/status', label: 'System Status', icon: Activity },
  ];

  const secondaryItems = [
    { label: 'Repositories', icon: FolderGit2, to: '/explore' },
    { label: 'Pull Requests', icon: GitPullRequest, to: '/dashboard' },
    { label: 'Issues', icon: CheckSquare, to: '/dashboard' },
    { label: 'Workflows & CI', icon: PlaySquare, to: '/status' },
    { label: 'Organizations', icon: Building2, to: '/dashboard' },
  ];

  return (
    <aside className="w-60 border-r border-forge-border bg-forge-surface flex flex-col justify-between p-3 select-none">
      <div className="space-y-6">
        {/* Main Navigation */}
        <div className="space-y-1">
          <div className="px-3 py-1 text-[11px] font-semibold text-forge-muted uppercase tracking-wider">
            Platform
          </div>
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  `flex items-center space-x-2.5 px-3 py-2 rounded-lg text-xs font-medium transition-colors ${
                    isActive
                      ? 'bg-blue-600/15 text-blue-400 border border-blue-500/20'
                      : 'text-forge-muted hover:text-forge-text hover:bg-forge-card'
                  }`
                }
              >
                <Icon className="w-4 h-4" />
                <span>{item.label}</span>
              </NavLink>
            );
          })}
        </div>

        {/* Workspaces & Resources */}
        <div className="space-y-1">
          <div className="px-3 py-1 text-[11px] font-semibold text-forge-muted uppercase tracking-wider">
            Collaboration
          </div>
          {secondaryItems.map((item, idx) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={idx}
                to={item.to}
                className="flex items-center space-x-2.5 px-3 py-2 rounded-lg text-xs font-medium text-forge-muted hover:text-forge-text hover:bg-forge-card transition-colors"
              >
                <Icon className="w-4 h-4" />
                <span>{item.label}</span>
              </NavLink>
            );
          })}
        </div>
      </div>

      {/* Footer Info */}
      <div className="pt-4 border-t border-forge-border/60 space-y-1">
        <NavLink
          to="/status"
          className={({ isActive }) =>
            `flex items-center justify-between px-3 py-2 rounded-lg text-xs font-medium transition-colors ${
              isActive
                ? 'bg-blue-600/15 text-blue-400 border border-blue-500/20'
                : 'text-forge-muted hover:text-forge-text hover:bg-forge-card'
            }`
          }
        >
          <div className="flex items-center space-x-2">
            <Activity className="w-3.5 h-3.5 text-emerald-400" />
            <span>Diagnostics</span>
          </div>
          <span className="text-[10px] text-forge-muted font-mono">v0.1.0</span>
        </NavLink>
      </div>
    </aside>
  );
};
