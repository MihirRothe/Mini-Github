import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, FolderGit2, Activity, Compass, PlusCircle, BookOpen, X } from 'lucide-react';

interface CommandPaletteProps {
  isOpen: boolean;
  onClose: () => void;
}

export const CommandPalette: React.FC<CommandPaletteProps> = ({ isOpen, onClose }) => {
  const [query, setQuery] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        isOpen ? onClose() : onClose(); // toggle handled by parent
      }
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const items = [
    { label: 'Go to Dashboard', icon: FolderGit2, action: () => navigate('/dashboard'), category: 'Navigation' },
    { label: 'Explore Repositories', icon: Compass, action: () => navigate('/explore'), category: 'Navigation' },
    { label: 'System Health & Telemetry', icon: Activity, action: () => navigate('/status'), category: 'Diagnostics' },
    { label: 'Create New Repository', icon: PlusCircle, action: () => navigate('/dashboard'), category: 'Actions' },
    { label: 'Read Architecture Docs', icon: BookOpen, action: () => window.open('https://github.com', '_blank'), category: 'Help' },
  ];

  const filteredItems = items.filter(i =>
    i.label.toLowerCase().includes(query.toLowerCase()) ||
    i.category.toLowerCase().includes(query.toLowerCase())
  );

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-24 bg-black/70 backdrop-blur-sm p-4 animate-fade-in">
      <div className="w-full max-w-xl bg-forge-surface border border-forge-border rounded-xl shadow-2xl overflow-hidden">
        {/* Search input header */}
        <div className="flex items-center px-4 py-3 border-b border-forge-border">
          <Search className="w-5 h-5 text-forge-muted mr-3" />
          <input
            autoFocus
            type="text"
            placeholder="Type a command or search (e.g. repo, health)..."
            value={query}
            onChange={e => setQuery(e.target.value)}
            className="w-full bg-transparent text-forge-text text-sm focus:outline-none placeholder:text-forge-muted"
          />
          <button
            onClick={onClose}
            className="text-forge-muted hover:text-forge-text p-1 rounded-md"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Command list */}
        <div className="max-h-80 overflow-y-auto p-2 divide-y divide-forge-border/40">
          {filteredItems.length === 0 ? (
            <div className="py-8 text-center text-sm text-forge-muted">
              No matching commands found for "{query}"
            </div>
          ) : (
            filteredItems.map((item, idx) => {
              const Icon = item.icon;
              return (
                <button
                  key={idx}
                  onClick={() => {
                    item.action();
                    onClose();
                  }}
                  className="w-full flex items-center justify-between px-3 py-2.5 rounded-lg text-sm text-forge-text hover:bg-forge-card hover:text-forge-accent transition-colors text-left"
                >
                  <div className="flex items-center space-x-3">
                    <Icon className="w-4 h-4 text-forge-muted" />
                    <span>{item.label}</span>
                  </div>
                  <span className="text-xs text-forge-muted bg-forge-bg px-2 py-0.5 rounded border border-forge-border">
                    {item.category}
                  </span>
                </button>
              );
            })
          )}
        </div>

        <div className="px-4 py-2 bg-forge-bg border-t border-forge-border text-xs text-forge-muted flex justify-between">
          <span>Navigate with mouse or keyboard</span>
          <span><kbd className="px-1 py-0.5 bg-forge-surface border border-forge-border rounded text-[10px]">ESC</kbd> to close</span>
        </div>
      </div>
    </div>
  );
};
