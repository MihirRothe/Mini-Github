import React, { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import {
  FileText,
  Copy,
  Check,
  ArrowLeft,
  Loader2,
  AlertCircle,
  Sparkles,
  TestTube
} from 'lucide-react';
import { AIExplainModal } from '../../components/ai/AIExplainModal';

interface BlobInfo {
  path: string;
  name: string;
  content: string;
  size: number;
  is_binary: boolean;
}

export const BlobViewPage: React.FC = () => {
  const { owner, repo: repoSlug, ref: urlRef, '*': filePath } = useParams<{
    owner: string;
    repo: string;
    ref: string;
    '*': string;
  }>();

  const [blob, setBlob] = useState<BlobInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [aiModalOpen, setAiModalOpen] = useState(false);
  const [aiModalMode, setAiModalMode] = useState<'explain' | 'tests'>('explain');

  useEffect(() => {
    if (!owner || !repoSlug || !urlRef || !filePath) return;
    setLoading(true);
    setError(null);

    fetch(`/api/v1/repos/${owner}/${repoSlug}/blob/${urlRef}/${filePath}`, {
      credentials: 'include',
    })
      .then(async (res) => {
        const data = await res.json();
        if (!res.ok) {
          throw new Error(data.error?.message || 'File not found.');
        }
        setBlob(data.blob);
      })
      .catch((err) => {
        setError(err.message || 'Failed to load file.');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [owner, repoSlug, urlRef, filePath]);

  const handleCopy = () => {
    if (!blob) return;
    navigator.clipboard.writeText(blob.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  if (loading) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading file...</p>
      </div>
    );
  }

  if (error || !blob) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <AlertCircle className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">File Not Found</h2>
        <p className="text-forge-muted mb-6">{error || 'The requested file could not be found.'}</p>
        <Link to={`/${owner}/${repoSlug}`} className="btn-secondary">Back to Repository</Link>
      </div>
    );
  }

  const lines = blob.content.split('\n');

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      {/* Breadcrumb path */}
      <div className="flex items-center gap-2 text-sm text-forge-muted">
        <Link to={`/${owner}/${repoSlug}`} className="text-forge-accent hover:underline flex items-center gap-1">
          <ArrowLeft className="w-4 h-4" />
          {repoSlug}
        </Link>
        <span>/</span>
        <span className="font-mono text-white">{filePath}</span>
      </div>

      {/* File Card Container */}
      <div className="card overflow-hidden">
        {/* File Toolbar */}
        <div className="p-3 px-4 border-b border-forge-border bg-forge-card/80 flex items-center justify-between text-xs">
          <div className="flex items-center gap-3 text-forge-muted font-mono">
            <FileText className="w-4 h-4 text-forge-accent" />
            <span className="font-semibold text-white">{blob.name}</span>
            <span>{lines.length} lines</span>
            <span>{blob.size} bytes</span>
          </div>

          <div className="flex items-center gap-2">
            {!blob.is_binary && (
              <>
                <button
                  onClick={() => {
                    setAiModalMode('explain');
                    setAiModalOpen(true);
                  }}
                  className="px-2.5 py-1 rounded bg-purple-500/10 hover:bg-purple-500/20 border border-purple-500/30 text-xs text-purple-300 transition-colors flex items-center gap-1.5"
                  title="Explain code structure with ForgeAI"
                >
                  <Sparkles className="w-3.5 h-3.5 text-purple-400" />
                  <span>Explain Code</span>
                </button>

                <button
                  onClick={() => {
                    setAiModalMode('tests');
                    setAiModalOpen(true);
                  }}
                  className="px-2.5 py-1 rounded bg-forge-bg hover:bg-forge-border border border-forge-border text-xs text-forge-text hover:text-white transition-colors flex items-center gap-1.5"
                  title="Generate unit test suite with ForgeAI"
                >
                  <TestTube className="w-3.5 h-3.5 text-blue-400" />
                  <span>Generate Tests</span>
                </button>
              </>
            )}

            <button
              onClick={handleCopy}
              className="px-2.5 py-1 rounded bg-forge-bg hover:bg-forge-border border border-forge-border text-xs text-forge-text transition-colors flex items-center gap-1"
            >
              {copied ? (
                <>
                  <Check className="w-3.5 h-3.5 text-emerald-400" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5" />
                  Copy raw
                </>
              )}
            </button>
          </div>
        </div>

        {/* Code View with Line Numbers */}
        {blob.is_binary ? (
          <div className="p-12 text-center text-xs text-forge-muted">
            Binary file cannot be displayed in text editor.
          </div>
        ) : (
          <div className="flex overflow-x-auto text-xs font-mono bg-forge-bg/60 p-4 leading-relaxed">
            {/* Line numbers column */}
            <div className="select-none text-right pr-4 text-forge-muted/40 border-r border-forge-border/40">
              {lines.map((_, i) => (
                <div key={i}>{i + 1}</div>
              ))}
            </div>

            {/* Code lines */}
            <div className="pl-4 text-forge-text whitespace-pre">
              {lines.map((line, i) => (
                <div key={i}>{line || '\n'}</div>
              ))}
            </div>
          </div>
        )}
      </div>

      {blob && !blob.is_binary && (
        <AIExplainModal
          isOpen={aiModalOpen}
          onClose={() => setAiModalOpen(false)}
          fileName={blob.name}
          code={blob.content}
          initialMode={aiModalMode}
        />
      )}
    </div>
  );
};
