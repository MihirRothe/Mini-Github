import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Search, FolderGit2, Star, GitFork, Lock, Globe, Plus, Loader2 } from 'lucide-react';
import { Repository } from '../types';

export const ExplorePage: React.FC = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'public' | 'private'>('all');

  const { data: repositories = [], isLoading, isError } = useQuery<Repository[]>({
    queryKey: ['explore-repositories'],
    queryFn: async () => {
      const res = await fetch('/api/v1/repos', { credentials: 'include' });
      if (!res.ok) throw new Error('Failed to fetch repositories');
      const data = await res.json();
      return data.repositories || [];
    },
  });

  const filteredRepos = repositories.filter((repo) => {
    const matchesSearch =
      repo.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      repo.owner_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (repo.description && repo.description.toLowerCase().includes(searchTerm.toLowerCase()));

    if (!matchesSearch) return false;
    if (filterType === 'all') return true;
    return repo.visibility === filterType;
  });

  return (
    <div className="space-y-6">
      <div className="flex flex-col md:flex-row md:items-center md:justify-between pb-4 border-b border-forge-border gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2">
            Explore Repositories
          </h1>
          <p className="text-sm text-forge-muted mt-1">
            Discover public repositories, open-source projects, and organization workspaces across ForgeHub.
          </p>
        </div>

        <Link
          to="/new"
          className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/20 transition-all self-start md:self-auto"
        >
          <Plus className="w-4 h-4" />
          <span>New Repository</span>
        </Link>
      </div>

      {/* Search & Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 w-4 h-4 text-forge-muted" />
          <input
            type="text"
            placeholder="Search repositories by name, owner, or description..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-9 pr-4 py-2 bg-forge-card border border-forge-border rounded-lg text-sm text-forge-text placeholder:text-forge-muted focus:outline-none focus:border-blue-500 transition-colors"
          />
        </div>

        <div className="flex items-center space-x-2">
          <div className="flex items-center bg-forge-card border border-forge-border rounded-lg p-1">
            <button
              onClick={() => setFilterType('all')}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterType === 'all'
                  ? 'bg-forge-surface text-white font-medium shadow-sm'
                  : 'text-forge-muted hover:text-forge-text'
              }`}
            >
              All
            </button>
            <button
              onClick={() => setFilterType('public')}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterType === 'public'
                  ? 'bg-forge-surface text-white font-medium shadow-sm'
                  : 'text-forge-muted hover:text-forge-text'
              }`}
            >
              Public
            </button>
            <button
              onClick={() => setFilterType('private')}
              className={`px-3 py-1 text-xs rounded-md transition-colors ${
                filterType === 'private'
                  ? 'bg-forge-surface text-white font-medium shadow-sm'
                  : 'text-forge-muted hover:text-forge-text'
              }`}
            >
              Private
            </button>
          </div>
        </div>
      </div>

      {/* Loading state */}
      {isLoading ? (
        <div className="glass-panel rounded-xl p-12 text-center space-y-3">
          <Loader2 className="w-8 h-8 text-blue-500 animate-spin mx-auto" />
          <p className="text-xs text-forge-muted">Loading repositories...</p>
        </div>
      ) : isError ? (
        <div className="glass-panel rounded-xl p-8 text-center text-rose-400 text-sm">
          Failed to load repositories from API.
        </div>
      ) : filteredRepos.length === 0 ? (
        <div className="glass-panel rounded-xl p-12 text-center space-y-4">
          <div className="w-14 h-14 rounded-2xl bg-forge-card border border-forge-border flex items-center justify-center mx-auto text-forge-muted">
            <FolderGit2 className="w-7 h-7" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-semibold text-white">No repositories match your query</h3>
            <p className="text-xs text-forge-muted max-w-md mx-auto">
              {repositories.length === 0
                ? 'There are currently no repositories published on this ForgeHub instance.'
                : 'Try adjusting your search terms or filter selection.'}
            </p>
          </div>
          {repositories.length === 0 && (
            <div className="pt-2">
              <Link
                to="/new"
                className="inline-flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow transition-all"
              >
                <Plus className="w-4 h-4" />
                <span>Create the first repository</span>
              </Link>
            </div>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {filteredRepos.map((repo) => (
            <Link
              key={repo.id}
              to={`/${repo.owner_name}/${repo.slug}`}
              className="glass-card p-4 rounded-xl space-y-3 hover:border-blue-500/50 transition-all block group"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2 min-w-0">
                  <span className="font-semibold text-sm text-blue-400 group-hover:underline truncate">
                    {repo.owner_name} / {repo.name}
                  </span>
                  <span className="flex items-center gap-1 text-[10px] font-medium px-2 py-0.5 rounded-full bg-forge-surface border border-forge-border text-forge-muted shrink-0">
                    {repo.visibility === 'public' ? <Globe className="w-3 h-3 text-emerald-400" /> : <Lock className="w-3 h-3 text-amber-400" />}
                    {repo.visibility}
                  </span>
                </div>
              </div>
              <p className="text-xs text-forge-muted line-clamp-2 min-h-[1.5rem]">
                {repo.description || 'No description provided.'}
              </p>
              <div className="flex items-center space-x-4 text-xs text-forge-muted pt-1">
                <span className="flex items-center space-x-1">
                  <Star className="w-3.5 h-3.5" />
                  <span>{repo.stars_count}</span>
                </span>
                <span className="flex items-center space-x-1">
                  <GitFork className="w-3.5 h-3.5" />
                  <span>{repo.forks_count}</span>
                </span>
                <span>Branch: <code className="font-mono text-[11px] text-forge-text">{repo.default_branch}</code></span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
};
