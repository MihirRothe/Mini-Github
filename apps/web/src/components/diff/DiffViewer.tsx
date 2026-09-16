import React, { useState } from 'react';
import { DiffResult, DiffFile } from '../../types';
import { FileCode, ChevronDown, ChevronRight, Plus, Minus } from 'lucide-react';

interface DiffViewerProps {
  diff: DiffResult;
}

export const DiffViewer: React.FC<DiffViewerProps> = ({ diff }) => {
  const [collapsedFiles, setCollapsedFiles] = useState<Record<string, boolean>>({});

  const toggleCollapse = (path: string) => {
    setCollapsedFiles((prev) => ({
      ...prev,
      [path]: !prev[path],
    }));
  };

  const collapseAll = () => {
    const next: Record<string, boolean> = {};
    diff.files.forEach((f) => {
      next[f.new_path || f.old_path] = true;
    });
    setCollapsedFiles(next);
  };

  const expandAll = () => {
    setCollapsedFiles({});
  };

  if (!diff.files || diff.files.length === 0) {
    return (
      <div className="card p-8 text-center text-forge-muted">
        <FileCode className="w-10 h-10 mx-auto mb-2 text-forge-muted/60" />
        <p className="font-medium text-white">There are no changes to display.</p>
        <p className="text-xs text-forge-muted mt-1">The compared branches are completely identical.</p>
      </div>
    );
  }

  const totalChanges = (diff.total_additions || 0) + (diff.total_deletions || 0);
  const addPercent = totalChanges > 0 ? ((diff.total_additions || 0) / totalChanges) * 100 : 0;

  return (
    <div className="space-y-6">
      {/* Summary Header */}
      <div className="card p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <span className="text-sm font-medium text-white">
            Showing <strong className="text-forge-accent">{diff.total_files} changed file{diff.total_files === 1 ? '' : 's'}</strong>
          </span>
          <div className="flex items-center gap-2 text-xs font-mono">
            <span className="text-emerald-400 font-semibold flex items-center gap-0.5">
              <Plus className="w-3.5 h-3.5" /> {diff.total_additions}
            </span>
            <span className="text-rose-400 font-semibold flex items-center gap-0.5">
              <Minus className="w-3.5 h-3.5" /> {diff.total_deletions}
            </span>
          </div>

          {/* Mini progress bar */}
          <div className="hidden md:flex w-28 h-2 rounded-full overflow-hidden bg-forge-border">
            <div
              className="bg-emerald-500 h-full transition-all"
              style={{ width: `${addPercent}%` }}
              title={`${diff.total_additions} additions`}
            />
            <div
              className="bg-rose-500 h-full transition-all"
              style={{ width: `${100 - addPercent}%` }}
              title={`${diff.total_deletions} deletions`}
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={expandAll}
            className="text-xs text-forge-muted hover:text-white px-2 py-1 rounded hover:bg-forge-bg transition"
          >
            Expand all
          </button>
          <span className="text-forge-border">|</span>
          <button
            onClick={collapseAll}
            className="text-xs text-forge-muted hover:text-white px-2 py-1 rounded hover:bg-forge-bg transition"
          >
            Collapse all
          </button>
        </div>
      </div>

      {/* Files List */}
      <div className="space-y-4">
        {diff.files.map((file) => (
          <DiffFileItem
            key={file.new_path || file.old_path}
            file={file}
            isCollapsed={!!collapsedFiles[file.new_path || file.old_path]}
            onToggle={() => toggleCollapse(file.new_path || file.old_path)}
          />
        ))}
      </div>
    </div>
  );
};

interface DiffFileItemProps {
  file: DiffFile;
  isCollapsed: boolean;
  onToggle: () => void;
}

const DiffFileItem: React.FC<DiffFileItemProps> = ({ file, isCollapsed, onToggle }) => {
  const filePath = file.new_path || file.old_path;

  const statusBadge = () => {
    switch (file.status) {
      case 'added':
        return <span className="text-[10px] px-1.5 py-0.5 rounded font-mono font-semibold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">ADDED</span>;
      case 'deleted':
        return <span className="text-[10px] px-1.5 py-0.5 rounded font-mono font-semibold bg-rose-500/10 text-rose-400 border border-rose-500/20">DELETED</span>;
      case 'renamed':
        return <span className="text-[10px] px-1.5 py-0.5 rounded font-mono font-semibold bg-blue-500/10 text-blue-400 border border-blue-500/20">RENAMED</span>;
      default:
        return <span className="text-[10px] px-1.5 py-0.5 rounded font-mono font-semibold bg-forge-bg text-forge-muted border border-forge-border">MODIFIED</span>;
    }
  };

  return (
    <div className="card overflow-hidden border border-forge-border/80">
      {/* File Header */}
      <div
        onClick={onToggle}
        className="px-4 py-2.5 bg-forge-card/90 hover:bg-forge-card border-b border-forge-border cursor-pointer flex items-center justify-between select-none transition"
      >
        <div className="flex items-center gap-2.5 min-w-0">
          <button type="button" className="text-forge-muted hover:text-white">
            {isCollapsed ? (
              <ChevronRight className="w-4 h-4" />
            ) : (
              <ChevronDown className="w-4 h-4" />
            )}
          </button>
          <FileCode className="w-4 h-4 text-forge-muted shrink-0" />
          <span className="font-mono text-xs text-white font-medium truncate">
            {filePath}
          </span>
          {file.old_path && file.old_path !== file.new_path && (
            <span className="text-xs text-forge-muted truncate font-mono">
              (was {file.old_path})
            </span>
          )}
          {statusBadge()}
        </div>

        <div className="flex items-center gap-3 text-xs font-mono shrink-0 ml-3">
          <span className="text-emerald-400 font-medium">+{file.additions}</span>
          <span className="text-rose-400 font-medium">-{file.deletions}</span>
        </div>
      </div>

      {/* File Hunks */}
      {!isCollapsed && (
        <div className="overflow-x-auto text-xs font-mono bg-forge-bg/60">
          {file.hunks && file.hunks.length > 0 ? (
            file.hunks.map((hunk, hunkIdx) => (
              <div key={hunkIdx} className="border-b last:border-b-0 border-forge-border/40">
                {/* Hunk Header */}
                <div className="px-4 py-1 bg-forge-card/40 text-forge-muted text-[11px] font-mono border-b border-forge-border/30">
                  {hunk.header}
                </div>

                {/* Hunk Lines */}
                <table className="w-full border-collapse">
                  <tbody>
                    {hunk.lines.map((line, lineIdx) => {
                      const isAdd = line.type === 'addition';
                      const isDel = line.type === 'deletion';

                      let rowClass = 'hover:bg-white/[0.02]';
                      let signClass = 'text-forge-muted/40';
                      let contentClass = 'text-forge-muted/90';

                      if (isAdd) {
                        rowClass = 'bg-emerald-950/30 hover:bg-emerald-950/40 text-emerald-200';
                        signClass = 'text-emerald-400 font-bold';
                        contentClass = 'text-emerald-100';
                      } else if (isDel) {
                        rowClass = 'bg-rose-950/30 hover:bg-rose-950/40 text-rose-200';
                        signClass = 'text-rose-400 font-bold';
                        contentClass = 'text-rose-100';
                      }

                      return (
                        <tr key={lineIdx} className={`leading-5 font-mono ${rowClass}`}>
                          {/* Old line number */}
                          <td className="w-12 px-2 py-0.5 text-right select-none text-[11px] text-forge-muted/40 border-r border-forge-border/30">
                            {line.old_line_no > 0 ? line.old_line_no : ''}
                          </td>
                          {/* New line number */}
                          <td className="w-12 px-2 py-0.5 text-right select-none text-[11px] text-forge-muted/40 border-r border-forge-border/30">
                            {line.new_line_no > 0 ? line.new_line_no : ''}
                          </td>
                          {/* Sign */}
                          <td className={`w-6 px-1.5 text-center select-none ${signClass}`}>
                            {isAdd ? '+' : isDel ? '-' : ' '}
                          </td>
                          {/* Code line */}
                          <td className={`px-2 py-0.5 whitespace-pre font-mono ${contentClass}`}>
                            {line.content || ' '}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            ))
          ) : (
            <div className="p-4 text-center text-xs text-forge-muted">
              Binary file or empty file changed without text hunks.
            </div>
          )}
        </div>
      )}
    </div>
  );
};
