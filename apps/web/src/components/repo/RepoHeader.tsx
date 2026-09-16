import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { Repository } from '../../types';
import {
  BookOpen,
  Code2,
  CircleDot,
  GitPullRequest,
  Globe,
  Lock,
  Terminal,
  Copy,
  Check
} from 'lucide-react';

interface RepoHeaderProps {
  repo: Repository;
  activeTab: 'code' | 'issues' | 'pulls' | 'settings';
  openIssuesCount?: number;
}

export const RepoHeader: React.FC<RepoHeaderProps> = ({
  repo,
  activeTab,
  openIssuesCount,
}) => {
  const [showCloneDropdown, setShowCloneDropdown] = useState(false);
  const [copiedClone, setCopiedClone] = useState(false);

  const cloneURL =
    repo.http_clone_url ||
    repo.clone_url ||
    `${window.location.origin}/${repo.owner_name}/${repo.slug}.git`;

  const handleCopyClone = () => {
    navigator.clipboard.writeText(cloneURL);
    setCopiedClone(true);
    setTimeout(() => setCopiedClone(false), 2000);
  };

  const tabs = [
    {
      id: 'code',
      label: 'Code',
      icon: Code2,
      href: `/${repo.owner_name}/${repo.slug}`,
    },
    {
      id: 'issues',
      label: 'Issues',
      icon: CircleDot,
      href: `/${repo.owner_name}/${repo.slug}/issues`,
      count: openIssuesCount,
    },
    {
      id: 'pulls',
      label: 'Pull Requests',
      icon: GitPullRequest,
      href: `/${repo.owner_name}/${repo.slug}/pulls`,
      count: 0,
    },
  ];

  return (
    <div className="border-b border-forge-border bg-forge-card/40 backdrop-blur">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-4">
          {/* Breadcrumb owner / repo */}
          <div className="flex items-center gap-2.5">
            <BookOpen className="w-5 h-5 text-forge-accent shrink-0" />
            <div className="flex items-center gap-1.5 text-lg sm:text-xl font-semibold text-white">
              <Link to={`/${repo.owner_name}`} className="text-forge-muted hover:text-white">
                {repo.owner_name}
              </Link>
              <span className="text-forge-muted/60">/</span>
              <Link to={`/${repo.owner_name}/${repo.slug}`} className="hover:underline">
                {repo.name}
              </Link>
            </div>

            <span className="ml-2 px-2 py-0.5 rounded-full text-[11px] font-semibold border flex items-center gap-1 bg-forge-bg text-forge-muted border-forge-border">
              {repo.visibility === 'public' ? (
                <>
                  <Globe className="w-3 h-3 text-emerald-400" />
                  Public
                </>
              ) : (
                <>
                  <Lock className="w-3 h-3 text-amber-400" />
                  Private
                </>
              )}
            </span>
          </div>

          {/* Code / Clone dropdown */}
          <div className="relative flex items-center gap-2">
            <button
              onClick={() => setShowCloneDropdown(!showCloneDropdown)}
              className="btn-primary text-xs flex items-center gap-2"
            >
              <Terminal className="w-4 h-4" />
              <span>Code / Clone</span>
            </button>

            {showCloneDropdown && (
              <div className="absolute right-0 top-10 mt-1 w-80 sm:w-96 bg-forge-surface border border-forge-border rounded-xl shadow-2xl p-4 z-50 animate-in fade-in zoom-in-95 duration-100">
                <div className="text-xs font-semibold text-white mb-2">Clone with HTTP</div>
                <div className="flex items-center rounded-lg bg-forge-bg border border-forge-border overflow-hidden">
                  <input
                    type="text"
                    readOnly
                    value={cloneURL}
                    className="w-full bg-transparent px-3 py-1.5 text-xs text-forge-text font-mono focus:outline-none"
                  />
                  <button
                    onClick={handleCopyClone}
                    className="px-3 py-1.5 bg-forge-card hover:bg-forge-border border-l border-forge-border text-xs text-forge-text transition-colors flex items-center gap-1"
                    title="Copy to clipboard"
                  >
                    {copiedClone ? (
                      <Check className="w-3.5 h-3.5 text-emerald-400" />
                    ) : (
                      <Copy className="w-3.5 h-3.5" />
                    )}
                  </button>
                </div>
                <p className="text-[11px] text-forge-muted mt-2">
                  Use your ForgeHub username and Personal Access Token to authenticate via CLI.
                </p>
              </div>
            )}
          </div>
        </div>

        {repo.description && (
          <p className="text-xs text-forge-muted pb-4">{repo.description}</p>
        )}

        {/* Tab Navigation */}
        <nav className="flex items-center gap-1 -mb-px overflow-x-auto">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <Link
                key={tab.id}
                to={tab.href}
                className={`flex items-center gap-2 px-3.5 py-2.5 text-xs font-medium border-b-2 transition-all select-none whitespace-nowrap ${
                  isActive
                    ? 'text-white border-blue-500 bg-forge-accent/5'
                    : 'text-forge-muted border-transparent hover:text-white hover:border-forge-border'
                }`}
              >
                <Icon className={`w-4 h-4 ${isActive ? 'text-blue-400' : 'text-forge-muted'}`} />
                <span>{tab.label}</span>
                {tab.count !== undefined && (
                  <span
                    className={`ml-1 px-1.5 py-0.2 rounded-full text-[10px] font-semibold ${
                      isActive
                        ? 'bg-blue-500/20 text-blue-300'
                        : 'bg-forge-surface text-forge-muted border border-forge-border'
                    }`}
                  >
                    {tab.count}
                  </span>
                )}
              </Link>
            );
          })}
        </nav>
      </div>
    </div>
  );
};
