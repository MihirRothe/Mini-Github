import React, { useEffect, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Organization } from '../../types';
import {
  BookOpen,
  Lock,
  Globe,
  AlertCircle,
  Loader2,
  FileCode
} from 'lucide-react';

export const NewRepoPage: React.FC = () => {
  const navigate = useNavigate();
  const { user } = useAuth();

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [visibility, setVisibility] = useState<'public' | 'private'>('public');
  const [initWithReadme, setInitWithReadme] = useState(true);

  // Owner selection: personal vs organizations
  const [ownerType, setOwnerType] = useState<'user' | 'org'>('user');
  const [ownerSlug, setOwnerSlug] = useState('');
  const [orgs, setOrgs] = useState<Organization[]>([]);

  const [isSlugManuallyEdited, setIsSlugManuallyEdited] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (user) {
      setOwnerSlug(user.username);
      // Fetch user's organizations
      fetch('/api/v1/orgs', { credentials: 'include' })
        .then((res) => (res.ok ? res.json() : { organizations: [] }))
        .then((data) => {
          setOrgs(data.organizations || []);
        })
        .catch(() => {});
    }
  }, [user]);

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setName(val);
    if (!isSlugManuallyEdited) {
      const generated = val
        .toLowerCase()
        .replace(/[^a-z0-9_.-]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 99);
      setSlug(generated);
    }
  };

  const handleSlugChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setIsSlugManuallyEdited(true);
    setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9_.-]/g, '').slice(0, 99));
  };

  const handleOwnerChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const val = e.target.value;
    if (val === user?.username) {
      setOwnerType('user');
      setOwnerSlug(val);
    } else {
      setOwnerType('org');
      setOwnerSlug(val);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Repository name is required.');
      return;
    }
    if (!slug.trim() || slug.length < 2) {
      setError('Repository name must be at least 2 characters.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await fetch('/api/v1/repos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim(),
          description: description.trim(),
          visibility,
          owner_type: ownerType,
          owner_slug: ownerSlug,
          default_branch: 'main',
          init_with_readme: initWithReadme,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to create repository.');
      }

      navigate(`/${data.repository.owner_name}/${data.repository.slug}`);
    } catch (err: any) {
      setError(err.message || 'An error occurred while creating the repository.');
    } finally {
      setLoading(false);
    }
  };

  if (!user) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <BookOpen className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Authentication Required</h2>
        <p className="text-forge-muted mb-6">You must be signed in to create a repository.</p>
        <Link to="/login" className="btn-primary">Sign In</Link>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto py-10 px-4 sm:px-6">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
          <BookOpen className="w-8 h-8 text-forge-accent" />
          Create a New Repository
        </h1>
        <p className="text-sm text-forge-muted mt-1">
          A repository contains all project files, revision history, commits, and branch workflows.
        </p>
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-950/40 border border-red-800/50 rounded-xl text-red-200 text-sm flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
          <span>{error}</span>
        </div>
      )}

      <form onSubmit={handleSubmit} className="card p-6 sm:p-8 space-y-6">
        {/* Owner & Repository Name */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="sm:col-span-1">
            <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
              Owner
            </label>
            <select
              value={ownerSlug}
              onChange={handleOwnerChange}
              className="input-field capitalize"
            >
              <option value={user.username}>{user.username} (personal)</option>
              {orgs.map((o) => (
                <option key={o.id} value={o.slug}>
                  {o.name} (@{o.slug})
                </option>
              ))}
            </select>
          </div>

          <div className="sm:col-span-2">
            <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
              Repository Name <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={handleNameChange}
              placeholder="e.g. cloud-orchestrator"
              className="input-field"
            />
          </div>
        </div>

        {/* Slug preview */}
        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
            Repository URL Path
          </label>
          <div className="flex items-center rounded-lg bg-forge-bg border border-forge-border overflow-hidden">
            <span className="px-3 py-2 text-xs text-forge-muted border-r border-forge-border bg-forge-card select-none">
              forgehub.local/{ownerSlug}/
            </span>
            <input
              type="text"
              required
              value={slug}
              onChange={handleSlugChange}
              placeholder="cloud-orchestrator"
              className="w-full bg-transparent px-3 py-2 text-sm text-white placeholder:text-forge-muted/50 focus:outline-none font-mono"
            />
          </div>
        </div>

        {/* Description */}
        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
            Description <span className="text-forge-muted/70 font-normal">(optional)</span>
          </label>
          <textarea
            rows={2}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Short summary of what this project does"
            className="input-field resize-none"
          />
        </div>

        {/* Visibility Radios */}
        <div className="space-y-3 pt-2">
          <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
            Visibility
          </label>

          <label
            className={`flex items-start gap-4 p-4 rounded-xl border cursor-pointer transition-colors ${
              visibility === 'public'
                ? 'bg-forge-accent/5 border-forge-accent'
                : 'bg-forge-card/40 border-forge-border hover:border-forge-subtle'
            }`}
          >
            <input
              type="radio"
              name="visibility"
              value="public"
              checked={visibility === 'public'}
              onChange={() => setVisibility('public')}
              className="mt-1 accent-blue-600"
            />
            <div>
              <div className="flex items-center gap-2 font-semibold text-sm text-white">
                <Globe className="w-4 h-4 text-forge-accent" />
                Public
              </div>
              <p className="text-xs text-forge-muted mt-0.5">
                Anyone on the network can view and clone this repository. You choose who can commit.
              </p>
            </div>
          </label>

          <label
            className={`flex items-start gap-4 p-4 rounded-xl border cursor-pointer transition-colors ${
              visibility === 'private'
                ? 'bg-forge-accent/5 border-forge-accent'
                : 'bg-forge-card/40 border-forge-border hover:border-forge-subtle'
            }`}
          >
            <input
              type="radio"
              name="visibility"
              value="private"
              checked={visibility === 'private'}
              onChange={() => setVisibility('private')}
              className="mt-1 accent-blue-600"
            />
            <div>
              <div className="flex items-center gap-2 font-semibold text-sm text-white">
                <Lock className="w-4 h-4 text-amber-400" />
                Private
              </div>
              <p className="text-xs text-forge-muted mt-0.5">
                Only you and people/teams you explicitly grant access can view or clone this repository.
              </p>
            </div>
          </label>
        </div>

        {/* Initialize with README */}
        <div className="pt-2 border-t border-forge-border">
          <label className="flex items-center gap-3 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={initWithReadme}
              onChange={(e) => setInitWithReadme(e.target.checked)}
              className="w-4 h-4 rounded bg-forge-bg border-forge-border text-blue-600 focus:ring-0 focus:ring-offset-0"
            />
            <div>
              <div className="text-sm font-medium text-white flex items-center gap-2">
                <FileCode className="w-4 h-4 text-forge-muted" />
                Initialize repository with a README
              </div>
              <p className="text-xs text-forge-muted">
                This lets you immediately clone the repository to your computer and browse files on the web.
              </p>
            </div>
          </label>
        </div>

        {/* Actions */}
        <div className="pt-4 border-t border-forge-border flex items-center justify-between">
          <Link to="/" className="btn-secondary text-sm">
            Cancel
          </Link>
          <button
            type="submit"
            disabled={loading}
            className="btn-primary text-sm flex items-center gap-2"
          >
            {loading ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Creating Repository...
              </>
            ) : (
              'Create Repository'
            )}
          </button>
        </div>
      </form>
    </div>
  );
};
