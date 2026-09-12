import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { 
  Building2, 
  ShieldCheck, 
  Globe, 
  MapPin, 
  AlertCircle,
  Loader2
} from 'lucide-react';

export const NewOrgPage: React.FC = () => {
  const navigate = useNavigate();
  const { user } = useAuth();

  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [website, setWebsite] = useState('');
  const [location, setLocation] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');

  const [isSlugManuallyEdited, setIsSlugManuallyEdited] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    setName(val);
    if (!isSlugManuallyEdited) {
      const generated = val
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 39);
      setSlug(generated);
    }
  };

  const handleSlugChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setIsSlugManuallyEdited(true);
    setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, '').slice(0, 39));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Organization name is required.');
      return;
    }
    if (!slug.trim() || slug.length < 3) {
      setError('Slug must be at least 3 characters long.');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await fetch('/api/v1/orgs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim(),
          description: description.trim(),
          website: website.trim(),
          location: location.trim(),
          avatar_url: avatarUrl.trim() || `https://api.dicebear.com/7.x/identicon/svg?seed=${slug.trim()}`,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to create organization.');
      }

      navigate(`/orgs/${data.organization.slug}`);
    } catch (err: any) {
      setError(err.message || 'An error occurred while creating the organization.');
    } finally {
      setLoading(false);
    }
  };

  if (!user) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <Building2 className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Authentication Required</h2>
        <p className="text-forge-muted mb-6">You must be signed in to create an organization.</p>
        <Link to="/login" className="btn-primary">Sign In</Link>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto py-10 px-4 sm:px-6">
      {/* Breadcrumb Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white flex items-center gap-3">
          <Building2 className="w-8 h-8 text-forge-accent" />
          Create a New Organization
        </h1>
        <p className="text-sm text-forge-muted mt-1">
          Organizations allow teams to collaborate across multiple repositories with centralized access control and role-based permissions.
        </p>
      </div>

      {error && (
        <div className="mb-6 p-4 bg-red-950/40 border border-red-800/50 rounded-xl text-red-200 text-sm flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
          <span>{error}</span>
        </div>
      )}

      <form onSubmit={handleSubmit} className="card p-6 sm:p-8 space-y-6">
        {/* Name & Slug */}
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
              Organization Display Name <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={handleNameChange}
              placeholder="e.g. Acme Corporation"
              className="input-field"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
              Organization URL Slug <span className="text-red-400">*</span>
            </label>
            <div className="flex items-center rounded-lg bg-forge-bg border border-forge-border focus-within:border-forge-accent focus-within:ring-1 focus-within:ring-forge-accent overflow-hidden">
              <span className="px-3 py-2 text-xs text-forge-muted border-r border-forge-border bg-forge-card select-none">
                forgehub.local/orgs/
              </span>
              <input
                type="text"
                required
                value={slug}
                onChange={handleSlugChange}
                placeholder="acme-corp"
                className="w-full bg-transparent px-3 py-2 text-sm text-white placeholder:text-forge-muted/50 focus:outline-none font-mono"
              />
            </div>
            <p className="text-xs text-forge-muted mt-1">
              Used in URLs and git remotes. Must be unique, 3-40 characters (alphanumeric, dashes, underscores).
            </p>
          </div>
        </div>

        {/* Description */}
        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
            Description
          </label>
          <div className="relative">
            <textarea
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="What does this organization build or collaborate on?"
              className="input-field resize-none"
            />
          </div>
        </div>

        {/* Optional Metadata */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-2">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5 flex items-center gap-1.5">
              <Globe className="w-3.5 h-3.5 text-forge-muted" /> Website
            </label>
            <input
              type="url"
              value={website}
              onChange={(e) => setWebsite(e.target.value)}
              placeholder="https://example.com"
              className="input-field"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5 flex items-center gap-1.5">
              <MapPin className="w-3.5 h-3.5 text-forge-muted" /> Location
            </label>
            <input
              type="text"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
              placeholder="e.g. San Francisco, CA"
              className="input-field"
            />
          </div>
        </div>

        {/* Avatar URL */}
        <div>
          <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
            Logo / Avatar URL (Optional)
          </label>
          <input
            type="url"
            value={avatarUrl}
            onChange={(e) => setAvatarUrl(e.target.value)}
            placeholder="https://example.com/logo.png"
            className="input-field"
          />
          <p className="text-xs text-forge-muted mt-1">
            Leave blank to automatically generate a unique vector geometric avatar.
          </p>
        </div>

        {/* Security / Role Notice Banner */}
        <div className="p-4 bg-forge-accent/5 border border-forge-accent/20 rounded-xl flex items-start gap-3">
          <ShieldCheck className="w-5 h-5 text-forge-accent shrink-0 mt-0.5" />
          <div className="text-xs text-forge-muted leading-relaxed">
            <span className="font-semibold text-white">Owner Privileges:</span> As the creator, you (<span className="text-forge-accent font-medium">{user.username}</span>) will automatically be assigned the <strong className="text-white">Owner</strong> role with complete administrative control over repositories, teams, and membership.
          </div>
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
                Creating Organization...
              </>
            ) : (
              'Create Organization'
            )}
          </button>
        </div>
      </form>
    </div>
  );
};
