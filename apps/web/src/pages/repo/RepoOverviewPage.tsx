import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { Repository } from '../../types';
import {
  BookOpen,
  GitBranch,
  GitCommit,
  Folder,
  FileText,
  Copy,
  Check,
  Globe,
  Lock,
  Loader2,
  Terminal,
  Clock,
  ArrowLeft
} from 'lucide-react';

interface BranchInfo {
  name: string;
  commit_hash: string;
  is_default: boolean;
}

interface CommitInfo {
  hash: string;
  short_hash: string;
  author_name: string;
  author_email: string;
  author_date: string;
  message: string;
}

interface TreeEntry {
  name: string;
  path: string;
  type: 'blob' | 'tree';
  mode: string;
  size: number;
}

interface BlobInfo {
  path: string;
  name: string;
  content: string;
  size: number;
}

export const RepoOverviewPage: React.FC = () => {
  const { owner, repo: repoSlug, ref: urlRef, '*': splatPath } = useParams<{
    owner: string;
    repo: string;
    ref?: string;
    '*'?: string;
  }>();
  const navigate = useNavigate();

  const [repo, setRepo] = useState<Repository | null>(null);
  const [branches, setBranches] = useState<BranchInfo[]>([]);
  const [activeRef, setActiveRef] = useState<string>('main');
  const [commits, setCommits] = useState<CommitInfo[]>([]);
  const [treeEntries, setTreeEntries] = useState<TreeEntry[]>([]);
  const [readme, setReadme] = useState<BlobInfo | null>(null);

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copiedClone, setCopiedClone] = useState(false);
  const [showCloneDropdown, setShowCloneDropdown] = useState(false);

  const subPath = splatPath || '';

  const fetchRepoData = async () => {
    if (!owner || !repoSlug) return;
    setLoading(true);
    setError(null);

    try {
      // 1. Fetch Repository Details
      const resRepo = await fetch(`/api/v1/repos/${owner}/${repoSlug}`, { credentials: 'include' });
      const dataRepo = await resRepo.json();
      if (!resRepo.ok) {
        throw new Error(dataRepo.error?.message || 'Repository not found.');
      }
      setRepo(dataRepo.repository);

      const defaultBranch = dataRepo.repository.default_branch || 'main';
      const currentRef = urlRef || defaultBranch;
      setActiveRef(currentRef);

      // 2. Fetch Branches
      const resBranches = await fetch(`/api/v1/repos/${owner}/${repoSlug}/branches`, {
        credentials: 'include',
      });
      if (resBranches.ok) {
        const dataB = await resBranches.json();
        setBranches(dataB.branches || []);
      }

      // 3. Fetch Commits on ref
      const resCommits = await fetch(`/api/v1/repos/${owner}/${repoSlug}/commits?ref=${currentRef}&limit=1`, {
        credentials: 'include',
      });
      if (resCommits.ok) {
        const dataC = await resCommits.json();
        setCommits(dataC.commits || []);
      }

      // 4. Fetch Tree
      const treeUrl = subPath
        ? `/api/v1/repos/${owner}/${repoSlug}/tree/${currentRef}/${subPath}`
        : `/api/v1/repos/${owner}/${repoSlug}/tree/${currentRef}`;
      const resTree = await fetch(treeUrl, { credentials: 'include' });
      if (resTree.ok) {
        const dataT = await resTree.json();
        setTreeEntries(dataT.tree || []);
      }

      // 5. Fetch Readme if at root
      if (!subPath) {
        const resReadme = await fetch(`/api/v1/repos/${owner}/${repoSlug}/readme?ref=${currentRef}`, {
          credentials: 'include',
        });
        if (resReadme.ok) {
          const dataR = await resReadme.json();
          setReadme(dataR.readme || null);
        } else {
          setReadme(null);
        }
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load repository.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRepoData();
  }, [owner, repoSlug, urlRef, splatPath]);

  const handleBranchChange = (newBranch: string) => {
    setActiveRef(newBranch);
    if (subPath) {
      navigate(`/${owner}/${repoSlug}/tree/${newBranch}/${subPath}`);
    } else {
      navigate(`/${owner}/${repoSlug}`);
    }
  };

  const handleCopyClone = () => {
    if (!repo) return;
    const url = repo.http_clone_url || repo.clone_url || `${window.location.origin}/${repo.owner_name}/${repo.slug}.git`;
    navigator.clipboard.writeText(url);
    setCopiedClone(true);
    setTimeout(() => setCopiedClone(false), 2000);
  };

  if (loading) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading repository...</p>
      </div>
    );
  }

  if (error || !repo) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <BookOpen className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Repository Not Found</h2>
        <p className="text-forge-muted mb-6">{error || 'This repository does not exist or is private.'}</p>
        <Link to="/" className="btn-secondary">Return Home</Link>
      </div>
    );
  }

  const latestCommit = commits[0];
  const cloneURL = repo.http_clone_url || repo.clone_url || `${window.location.origin}/${repo.owner_name}/${repo.slug}.git`;

  return (
    <div className="min-h-screen bg-forge-bg pb-16">
      {/* Top Header Navigation */}
      <div className="border-b border-forge-border bg-forge-card/40 backdrop-blur">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
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

            {/* Actions: Clone & Code */}
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
            <p className="text-xs text-forge-muted mt-2">{repo.description}</p>
          )}
        </div>
      </div>

      {/* Main Content Area */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
        {/* Controls Bar: Branch Selector & Path Breadcrumbs */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            {/* Branch Selector */}
            <div className="relative flex items-center gap-1.5 bg-forge-card border border-forge-border rounded-lg px-2.5 py-1 text-xs text-white">
              <GitBranch className="w-3.5 h-3.5 text-forge-muted" />
              <select
                value={activeRef}
                onChange={(e) => handleBranchChange(e.target.value)}
                className="bg-transparent text-white text-xs font-medium focus:outline-none pr-2 cursor-pointer font-mono"
              >
                {branches.map((b) => (
                  <option key={b.name} value={b.name} className="bg-forge-card text-white">
                    {b.name} {b.is_default ? '(default)' : ''}
                  </option>
                ))}
              </select>
            </div>

            {/* Path Breadcrumbs when inside folder */}
            {subPath && (
              <div className="flex items-center gap-1.5 text-xs text-forge-muted">
                <Link
                  to={`/${owner}/${repoSlug}`}
                  className="text-forge-accent hover:underline flex items-center gap-1"
                >
                  <ArrowLeft className="w-3.5 h-3.5" />
                  {repo.name}
                </Link>
                <span>/</span>
                <span className="font-mono text-white">{subPath}</span>
              </div>
            )}
          </div>

          {/* Commits count link */}
          <Link
            to={`/${owner}/${repoSlug}/commits`}
            className="flex items-center gap-1.5 text-xs text-forge-muted hover:text-white transition-colors"
          >
            <GitCommit className="w-3.5 h-3.5" />
            <span>Commits History</span>
          </Link>
        </div>

        {/* Empty Repo Setup Instructions */}
        {treeEntries.length === 0 && !latestCommit ? (
          <div className="card p-8 space-y-6">
            <div className="text-center max-w-lg mx-auto">
              <BookOpen className="w-12 h-12 mx-auto text-forge-muted/50 mb-3" />
              <h3 className="text-lg font-bold text-white mb-1">Quick Setup — Get Started</h3>
              <p className="text-xs text-forge-muted">
                This is a brand new empty repository. Push code from your computer using the command line below.
              </p>
            </div>

            <div className="space-y-3 max-w-2xl mx-auto">
              <div className="text-xs font-semibold text-white">Create a new repository on the command line</div>
              <pre className="p-4 rounded-xl bg-forge-bg border border-forge-border text-xs text-forge-text font-mono overflow-x-auto leading-relaxed">
                echo "# {repo.name}" &gt;&gt; README.md{'\n'}
                git init{'\n'}
                git add README.md{'\n'}
                git commit -m "first commit"{'\n'}
                git branch -M main{'\n'}
                git remote add origin {cloneURL}{'\n'}
                git push -u origin main
              </pre>
            </div>

            <div className="space-y-3 max-w-2xl mx-auto">
              <div className="text-xs font-semibold text-white">Push an existing repository from the command line</div>
              <pre className="p-4 rounded-xl bg-forge-bg border border-forge-border text-xs text-forge-text font-mono overflow-x-auto leading-relaxed">
                git remote add origin {cloneURL}{'\n'}
                git branch -M main{'\n'}
                git push -u origin main
              </pre>
            </div>
          </div>
        ) : (
          <>
            {/* Latest Commit Bar */}
            {latestCommit && (
              <div className="card p-3 sm:px-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs bg-forge-card/80">
                <div className="flex items-center gap-2.5">
                  <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-blue-600 to-indigo-600 flex items-center justify-center text-[10px] font-bold text-white uppercase">
                    {latestCommit.author_name.slice(0, 1)}
                  </div>
                  <span className="font-semibold text-white">{latestCommit.author_name}</span>
                  <span className="text-forge-muted truncate max-w-md">{latestCommit.message}</span>
                </div>

                <div className="flex items-center gap-3 text-forge-muted self-end sm:self-auto">
                  <span className="flex items-center gap-1 font-mono text-[11px]">
                    <Clock className="w-3 h-3" />
                    {new Date(latestCommit.author_date).toLocaleDateString()}
                  </span>
                  <span className="font-mono bg-forge-bg px-2 py-0.5 rounded border border-forge-border text-[11px] text-forge-text">
                    {latestCommit.short_hash}
                  </span>
                </div>
              </div>
            )}

            {/* File Tree Browser */}
            <div className="card divide-y divide-forge-border overflow-hidden">
              {/* Back to parent directory if inside folder */}
              {subPath && (
                <Link
                  to={
                    subPath.includes('/')
                      ? `/${owner}/${repoSlug}/tree/${activeRef}/${subPath.split('/').slice(0, -1).join('/')}`
                      : `/${owner}/${repoSlug}`
                  }
                  className="p-3 px-4 flex items-center gap-3 text-xs font-mono text-forge-accent hover:bg-forge-card/50 transition-colors"
                >
                  <Folder className="w-4 h-4 text-forge-accent" />
                  <span>..</span>
                </Link>
              )}

              {treeEntries.map((entry) => (
                <div
                  key={entry.path}
                  className="p-3 px-4 flex items-center justify-between hover:bg-forge-card/40 transition-colors text-xs font-mono"
                >
                  <div className="flex items-center gap-3">
                    {entry.type === 'tree' ? (
                      <Folder className="w-4 h-4 text-blue-400 shrink-0" />
                    ) : (
                      <FileText className="w-4 h-4 text-forge-muted shrink-0" />
                    )}

                    {entry.type === 'tree' ? (
                      <Link
                        to={`/${owner}/${repoSlug}/tree/${activeRef}/${entry.path}`}
                        className="text-white hover:text-forge-accent hover:underline"
                      >
                        {entry.name}
                      </Link>
                    ) : (
                      <Link
                        to={`/${owner}/${repoSlug}/blob/${activeRef}/${entry.path}`}
                        className="text-white hover:text-forge-accent hover:underline"
                      >
                        {entry.name}
                      </Link>
                    )}
                  </div>

                  <span className="text-forge-muted text-[11px]">
                    {entry.type === 'blob' && `${entry.size} B`}
                  </span>
                </div>
              ))}
            </div>

            {/* README Preview Card */}
            {readme && (
              <div className="card overflow-hidden">
                <div className="px-5 py-3 border-b border-forge-border bg-forge-card/60 flex items-center gap-2 text-xs font-semibold text-white">
                  <BookOpen className="w-4 h-4 text-forge-accent" />
                  <span>{readme.name}</span>
                </div>
                <div className="p-6 prose prose-invert max-w-none text-sm leading-relaxed whitespace-pre-wrap font-sans">
                  {readme.content}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
