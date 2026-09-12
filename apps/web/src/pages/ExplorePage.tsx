import React, { useState } from 'react';
import { Search, FolderGit2, Star, GitFork, Lock, Globe, Plus } from 'lucide-react';
import { Repository } from '../types';

export const ExplorePage: React.FC = () => {
  const [searchTerm, setSearchTerm] = useState('');
  const [filterType, setFilterType] = useState<'all' | 'public' | 'private'>('all');

  // Empty repository collection initially until Phase 4 (Git Repositories)
  const repositories: Repository[] = [];

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

        <button
          className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold shadow-lg shadow-blue-600/20 transition-all self-start md:self-auto"
        >
          <Plus className="w-4 h-4" />
          <span>New Repository</span>
        </button>
      </div>

      {/* Search & Filters */}
      <div className="flex flex-col sm:flex-row gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 w-4 h-4 text-forge-muted" />
          <input
            type="text"
            placeholder="Search repositories by name or topic..."
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

      {/* Repositories List / Empty State */}
      {repositories.length === 0 ? (
        <div className="glass-panel rounded-xl p-12 text-center space-y-4">
          <div className="w-14 h-14 rounded-2xl bg-forge-card border border-forge-border flex items-center justify-center mx-auto text-forge-muted">
            <FolderGit2 className="w-7 h-7" />
          </div>
          <div className="space-y-1">
            <h3 className="text-base font-semibold text-white">No repositories match your query</h3>
            <p className="text-xs text-forge-muted max-w-md mx-auto">
              There are currently no repositories published on this ForgeHub instance. Repositories created in Phase 4 will appear here.
            </p>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {repositories.map((repo) => (
            <div key={repo.id} className="glass-card p-4 rounded-xl space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2">
                  <span className="font-semibold text-sm text-blue-400 hover:underline cursor-pointer">
                    {repo.owner_name} / {repo.name}
                  </span>
                  <span className="flex items-center gap-1 text-[10px] font-medium px-2 py-0.5 rounded-full bg-forge-surface border border-forge-border text-forge-muted">
                    {repo.visibility === 'public' ? <Globe className="w-3 h-3" /> : <Lock className="w-3 h-3" />}
                    {repo.visibility}
                  </span>
                </div>
              </div>
              <p className="text-xs text-forge-muted line-clamp-2">{repo.description}</p>
              <div className="flex items-center space-x-4 text-xs text-forge-muted">
                <span className="flex items-center space-x-1">
                  <Star className="w-3.5 h-3.5" />
                  <span>{repo.stars_count}</span>
                </span>
                <span className="flex items-center space-x-1">
                  <GitFork className="w-3.5 h-3.5" />
                  <span>{repo.forks_count}</span>
                </span>
                <span>Updated {repo.updated_at}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
