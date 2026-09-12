import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Organization, OrgMember, Team, OrgRole } from '../../types';
import {
  Building2,
  Users,
  Code2,
  Settings,
  Globe,
  MapPin,
  Plus,
  UserPlus,
  Trash2,
  AlertTriangle,
  Check,
  Loader2
} from 'lucide-react';

type Tab = 'repositories' | 'teams' | 'people' | 'settings';

export const OrgOverviewPage: React.FC = () => {
  const { org: orgSlug } = useParams<{ org: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [org, setOrg] = useState<Organization | null>(null);
  const [members, setMembers] = useState<OrgMember[]>([]);
  const [teams, setTeams] = useState<Team[]>([]);
  const [activeTab, setActiveTab] = useState<Tab>('repositories');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Invite Member Modal State
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [inviteUsername, setInviteUsername] = useState('');
  const [inviteRole, setInviteRole] = useState<OrgRole>('member');
  const [inviteLoading, setInviteLoading] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  // Create Team Modal State
  const [showTeamModal, setShowTeamModal] = useState(false);
  const [teamName, setTeamName] = useState('');
  const [teamSlug, setTeamSlug] = useState('');
  const [teamDesc, setTeamDesc] = useState('');
  const [teamPrivacy, setTeamPrivacy] = useState<'visible' | 'secret'>('visible');
  const [teamLoading, setTeamLoading] = useState(false);
  const [teamError, setTeamError] = useState<string | null>(null);

  // Settings form state
  const [settingsName, setSettingsName] = useState('');
  const [settingsDesc, setSettingsDesc] = useState('');
  const [settingsWebsite, setSettingsWebsite] = useState('');
  const [settingsLocation, setSettingsLocation] = useState('');
  const [settingsLoading, setSettingsLoading] = useState(false);
  const [settingsMsg, setSettingsMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchOrgData = async () => {
    if (!orgSlug) return;
    setLoading(true);
    setError(null);

    try {
      // 1. Fetch Org details
      const res = await fetch(`/api/v1/orgs/${orgSlug}`, { credentials: 'include' });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Organization not found.');
      }
      setOrg(data.organization);
      setSettingsName(data.organization.name);
      setSettingsDesc(data.organization.description || '');
      setSettingsWebsite(data.organization.website || '');
      setSettingsLocation(data.organization.location || '');

      // 2. Fetch Members
      const resMembers = await fetch(`/api/v1/orgs/${orgSlug}/members`, { credentials: 'include' });
      if (resMembers.ok) {
        const memData = await resMembers.json();
        setMembers(memData.members || []);
      }

      // 3. Fetch Teams
      const resTeams = await fetch(`/api/v1/orgs/${orgSlug}/teams`, { credentials: 'include' });
      if (resTeams.ok) {
        const teamData = await resTeams.json();
        setTeams(teamData.teams || []);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load organization.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOrgData();
  }, [orgSlug]);

  const isOwner = org?.role === 'owner' || user?.is_admin;
  const isAdmin = isOwner || org?.role === 'admin';

  // Handle Invite Member
  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteUsername.trim()) return;
    setInviteLoading(true);
    setInviteError(null);

    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/members`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          username: inviteUsername.trim(),
          role: inviteRole,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to add member.');
      }
      setMembers((prev) => [...prev, data.member]);
      setShowInviteModal(false);
      setInviteUsername('');
    } catch (err: any) {
      setInviteError(err.message);
    } finally {
      setInviteLoading(false);
    }
  };

  // Handle Member Role Update
  const handleRoleChange = async (username: string, newRole: OrgRole) => {
    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/members/${username}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ role: newRole }),
      });
      if (!res.ok) {
        const data = await res.json();
        alert(data.error?.message || 'Failed to update member role.');
        return;
      }
      setMembers((prev) =>
        prev.map((m) => (m.username === username ? { ...m, role: newRole } : m))
      );
    } catch (err: any) {
      alert(err.message);
    }
  };

  // Handle Remove Member
  const handleRemoveMember = async (username: string) => {
    if (!confirm(`Are you sure you want to remove @${username} from ${org?.name}?`)) return;
    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/members/${username}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok) {
        const data = await res.json();
        alert(data.error?.message || 'Failed to remove member.');
        return;
      }
      setMembers((prev) => prev.filter((m) => m.username !== username));
    } catch (err: any) {
      alert(err.message);
    }
  };

  // Handle Create Team
  const handleCreateTeam = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!teamName.trim() || !teamSlug.trim()) return;
    setTeamLoading(true);
    setTeamError(null);

    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/teams`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          name: teamName.trim(),
          slug: teamSlug.trim(),
          description: teamDesc.trim(),
          privacy: teamPrivacy,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to create team.');
      }
      setTeams((prev) => [...prev, data.team]);
      setShowTeamModal(false);
      setTeamName('');
      setTeamSlug('');
      setTeamDesc('');
    } catch (err: any) {
      setTeamError(err.message);
    } finally {
      setTeamLoading(false);
    }
  };

  // Handle Update Org Settings
  const handleUpdateSettings = async (e: React.FormEvent) => {
    e.preventDefault();
    setSettingsLoading(true);
    setSettingsMsg(null);

    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          name: settingsName.trim(),
          description: settingsDesc.trim(),
          website: settingsWebsite.trim(),
          location: settingsLocation.trim(),
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to update organization.');
      }
      setOrg(data.organization);
      setSettingsMsg({ type: 'success', text: 'Organization settings updated successfully!' });
    } catch (err: any) {
      setSettingsMsg({ type: 'error', text: err.message });
    } finally {
      setSettingsLoading(false);
    }
  };

  // Handle Delete Org
  const handleDeleteOrg = async () => {
    const confirmation = prompt(
      `To confirm deletion, type "${orgSlug}" below:`
    );
    if (confirmation !== orgSlug) return;

    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok) {
        const data = await res.json();
        alert(data.error?.message || 'Failed to delete organization.');
        return;
      }
      navigate('/');
    } catch (err: any) {
      alert(err.message);
    }
  };

  if (loading) {
    return (
      <div className="max-w-6xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading organization details...</p>
      </div>
    );
  }

  if (error || !org) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <Building2 className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Organization Not Found</h2>
        <p className="text-forge-muted mb-6">{error || "The organization you are looking for does not exist."}</p>
        <Link to="/" className="btn-secondary">Return Home</Link>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-forge-bg">
      {/* Org Header Profile Banner */}
      <div className="border-b border-forge-border bg-forge-card/40 backdrop-blur">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
            <div className="flex items-center gap-5">
              <img
                src={org.avatar_url || `https://api.dicebear.com/7.x/identicon/svg?seed=${org.slug}`}
                alt={org.name}
                className="w-20 h-20 rounded-2xl bg-forge-bg border-2 border-forge-border shadow-lg object-cover"
              />
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-2xl sm:text-3xl font-bold text-white">{org.name}</h1>
                  {org.role && (
                    <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-forge-accent/20 text-forge-accent border border-forge-accent/30 capitalize">
                      {org.role}
                    </span>
                  )}
                </div>
                <div className="text-sm font-mono text-forge-muted mt-0.5">@{org.slug}</div>
                {org.description && (
                  <p className="text-sm text-forge-muted mt-2 max-w-2xl">{org.description}</p>
                )}
                <div className="flex flex-wrap items-center gap-4 mt-3 text-xs text-forge-muted">
                  {org.location && (
                    <span className="flex items-center gap-1">
                      <MapPin className="w-3.5 h-3.5" />
                      {org.location}
                    </span>
                  )}
                  {org.website && (
                    <a
                      href={org.website.startsWith('http') ? org.website : `https://${org.website}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex items-center gap-1 text-forge-accent hover:underline"
                    >
                      <Globe className="w-3.5 h-3.5" />
                      {org.website.replace(/^https?:\/\//, '')}
                    </a>
                  )}
                  <span>Created {new Date(org.created_at).toLocaleDateString()}</span>
                </div>
              </div>
            </div>

            {/* Quick Actions for Admins */}
            {isAdmin && (
              <div className="flex items-center gap-3">
                <button
                  onClick={() => setShowInviteModal(true)}
                  className="btn-secondary text-xs flex items-center gap-1.5"
                >
                  <UserPlus className="w-4 h-4" />
                  Invite Member
                </button>
                <button
                  onClick={() => setShowTeamModal(true)}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  <Plus className="w-4 h-4" />
                  New Team
                </button>
              </div>
            )}
          </div>

          {/* Tab Navigation */}
          <div className="flex items-center gap-6 mt-8 border-t border-forge-border/60 pt-4 overflow-x-auto">
            <button
              onClick={() => setActiveTab('repositories')}
              className={`flex items-center gap-2 text-sm font-medium pb-1 border-b-2 transition-colors whitespace-nowrap ${
                activeTab === 'repositories'
                  ? 'border-forge-accent text-white'
                  : 'border-transparent text-forge-muted hover:text-white'
              }`}
            >
              <Code2 className="w-4 h-4" />
              Repositories
              <span className="px-2 py-0.2 rounded-full text-xs bg-forge-border text-forge-muted">
                {org.repo_count ?? 0}
              </span>
            </button>

            <button
              onClick={() => setActiveTab('teams')}
              className={`flex items-center gap-2 text-sm font-medium pb-1 border-b-2 transition-colors whitespace-nowrap ${
                activeTab === 'teams'
                  ? 'border-forge-accent text-white'
                  : 'border-transparent text-forge-muted hover:text-white'
              }`}
            >
              <Users className="w-4 h-4" />
              Teams
              <span className="px-2 py-0.2 rounded-full text-xs bg-forge-border text-forge-muted">
                {teams.length}
              </span>
            </button>

            <button
              onClick={() => setActiveTab('people')}
              className={`flex items-center gap-2 text-sm font-medium pb-1 border-b-2 transition-colors whitespace-nowrap ${
                activeTab === 'people'
                  ? 'border-forge-accent text-white'
                  : 'border-transparent text-forge-muted hover:text-white'
              }`}
            >
              <Users className="w-4 h-4" />
              People
              <span className="px-2 py-0.2 rounded-full text-xs bg-forge-border text-forge-muted">
                {members.length}
              </span>
            </button>

            {isAdmin && (
              <button
                onClick={() => setActiveTab('settings')}
                className={`flex items-center gap-2 text-sm font-medium pb-1 border-b-2 transition-colors whitespace-nowrap ${
                  activeTab === 'settings'
                    ? 'border-forge-accent text-white'
                    : 'border-transparent text-forge-muted hover:text-white'
                }`}
              >
                <Settings className="w-4 h-4" />
                Settings
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* TAB 1: REPOSITORIES */}
        {activeTab === 'repositories' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-white">Repositories</h2>
              {isAdmin && (
                <button
                  disabled
                  title="Coming in Phase 4 (Git Hosting)"
                  className="btn-secondary text-xs opacity-60 cursor-not-allowed flex items-center gap-1.5"
                >
                  <Plus className="w-4 h-4" />
                  New Repository (Phase 4)
                </button>
              )}
            </div>

            <div className="card p-8 text-center">
              <Code2 className="w-12 h-12 mx-auto text-forge-muted/60 mb-3" />
              <h3 className="text-base font-semibold text-white mb-1">No Repositories Hosted Yet</h3>
              <p className="text-xs text-forge-muted max-w-md mx-auto mb-4">
                Repositories for <span className="text-white font-mono">{org.slug}</span> will be enabled in Phase 4 (Git Storage & Smart HTTP Engine).
              </p>
            </div>
          </div>
        )}

        {/* TAB 2: TEAMS */}
        {activeTab === 'teams' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-white">Teams</h2>
                <p className="text-xs text-forge-muted">
                  Teams are groups of organization members that reflect your company or group structure with cascading permissions.
                </p>
              </div>
              {isAdmin && (
                <button
                  onClick={() => setShowTeamModal(true)}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  <Plus className="w-4 h-4" />
                  New Team
                </button>
              )}
            </div>

            {teams.length === 0 ? (
              <div className="card p-8 text-center">
                <Users className="w-12 h-12 mx-auto text-forge-muted/60 mb-3" />
                <h3 className="text-base font-semibold text-white mb-1">No Teams Created</h3>
                <p className="text-xs text-forge-muted max-w-md mx-auto mb-4">
                  Create teams like @engineering, @design, or @security to grant group-level permissions to repositories.
                </p>
                {isAdmin && (
                  <button onClick={() => setShowTeamModal(true)} className="btn-primary text-xs">
                    Create First Team
                  </button>
                )}
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {teams.map((team) => (
                  <div key={team.id} className="card p-5 hover:border-forge-accent/40 transition-colors">
                    <div className="flex items-start justify-between mb-3">
                      <div>
                        <h3 className="text-base font-semibold text-white">{team.name}</h3>
                        <span className="text-xs font-mono text-forge-muted">@{org.slug}/{team.slug}</span>
                      </div>
                      <span className="px-2 py-0.5 rounded text-[10px] font-semibold uppercase bg-forge-border text-forge-muted">
                        {team.privacy}
                      </span>
                    </div>

                    <p className="text-xs text-forge-muted mb-4 line-clamp-2">
                      {team.description || 'No description provided.'}
                    </p>

                    <div className="flex items-center justify-between pt-3 border-t border-forge-border text-xs text-forge-muted">
                      <span className="flex items-center gap-1">
                        <Users className="w-3.5 h-3.5" />
                        {team.member_count} {team.member_count === 1 ? 'member' : 'members'}
                      </span>
                      <Link
                        to={`/orgs/${org.slug}/teams/${team.slug}`}
                        className="text-forge-accent hover:underline font-medium"
                      >
                        View Team &rarr;
                      </Link>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* TAB 3: PEOPLE (MEMBERS) */}
        {activeTab === 'people' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-white">Organization Members</h2>
                <p className="text-xs text-forge-muted">
                  Users with membership in {org.name}. Owners have full access to manage settings, billing, teams, and repositories.
                </p>
              </div>
              {isAdmin && (
                <button
                  onClick={() => setShowInviteModal(true)}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  <UserPlus className="w-4 h-4" />
                  Invite Member
                </button>
              )}
            </div>

            <div className="card divide-y divide-forge-border overflow-hidden">
              {members.map((member) => (
                <div key={member.id} className="p-4 sm:px-6 flex items-center justify-between gap-4">
                  <div className="flex items-center gap-3">
                    <img
                      src={member.avatar_url || `https://api.dicebear.com/7.x/identicon/svg?seed=${member.username}`}
                      alt={member.username}
                      className="w-10 h-10 rounded-full bg-forge-bg border border-forge-border object-cover"
                    />
                    <div>
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/${member.username}`}
                          className="text-sm font-semibold text-white hover:text-forge-accent"
                        >
                          {member.display_name || member.username}
                        </Link>
                        <span className="text-xs font-mono text-forge-muted">@{member.username}</span>
                      </div>
                      <span className="text-xs text-forge-muted">{member.email}</span>
                    </div>
                  </div>

                  <div className="flex items-center gap-4">
                    {/* Role selector for Owner/Admin, or Static Badge */}
                    {isOwner && member.username !== user?.username ? (
                      <select
                        value={member.role}
                        onChange={(e) => handleRoleChange(member.username, e.target.value as OrgRole)}
                        className="bg-forge-bg border border-forge-border rounded-lg text-xs text-white px-2.5 py-1 focus:outline-none focus:border-forge-accent capitalize"
                      >
                        <option value="member">Member</option>
                        <option value="admin">Admin</option>
                        <option value="owner">Owner</option>
                        <option value="billing_manager">Billing Manager</option>
                      </select>
                    ) : (
                      <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-forge-border text-forge-text capitalize">
                        {member.role}
                      </span>
                    )}

                    {/* Remove Action */}
                    {(isOwner || (isAdmin && member.role === 'member') || member.username === user?.username) && (
                      <button
                        onClick={() => handleRemoveMember(member.username)}
                        title={member.username === user?.username ? "Leave Organization" : "Remove Member"}
                        className="p-1.5 text-forge-muted hover:text-red-400 rounded transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* TAB 4: SETTINGS */}
        {activeTab === 'settings' && isAdmin && (
          <div className="space-y-8 max-w-3xl">
            {settingsMsg && (
              <div
                className={`p-4 rounded-xl text-sm flex items-center gap-3 ${
                  settingsMsg.type === 'success'
                    ? 'bg-emerald-950/40 border border-emerald-800/50 text-emerald-200'
                    : 'bg-red-950/40 border border-red-800/50 text-red-200'
                }`}
              >
                {settingsMsg.type === 'success' ? (
                  <Check className="w-5 h-5 text-emerald-400 shrink-0" />
                ) : (
                  <AlertTriangle className="w-5 h-5 text-red-400 shrink-0" />
                )}
                <span>{settingsMsg.text}</span>
              </div>
            )}

            <form onSubmit={handleUpdateSettings} className="card p-6 sm:p-8 space-y-5">
              <h3 className="text-base font-semibold text-white border-b border-forge-border pb-3">
                General Information
              </h3>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
                  Organization Name
                </label>
                <input
                  type="text"
                  required
                  value={settingsName}
                  onChange={(e) => setSettingsName(e.target.value)}
                  className="input-field"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
                  Description
                </label>
                <textarea
                  rows={3}
                  value={settingsDesc}
                  onChange={(e) => setSettingsDesc(e.target.value)}
                  className="input-field resize-none"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
                    Website
                  </label>
                  <input
                    type="url"
                    value={settingsWebsite}
                    onChange={(e) => setSettingsWebsite(e.target.value)}
                    className="input-field"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-forge-muted mb-1.5">
                    Location
                  </label>
                  <input
                    type="text"
                    value={settingsLocation}
                    onChange={(e) => setSettingsLocation(e.target.value)}
                    className="input-field"
                  />
                </div>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={settingsLoading}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  {settingsLoading ? 'Saving...' : 'Save Organization Changes'}
                </button>
              </div>
            </form>

            {/* Danger Zone (Owner Only) */}
            {isOwner && (
              <div className="card p-6 sm:p-8 border-red-900/40 bg-red-950/10 space-y-4">
                <h3 className="text-base font-semibold text-red-400 flex items-center gap-2 border-b border-red-900/30 pb-3">
                  <AlertTriangle className="w-5 h-5 text-red-400" />
                  Danger Zone
                </h3>
                <p className="text-xs text-forge-muted leading-relaxed">
                  Deleting this organization will permanently remove all teams, repository permissions, and membership records. This action cannot be undone.
                </p>
                <button
                  onClick={handleDeleteOrg}
                  className="px-4 py-2 rounded-lg bg-red-900/30 hover:bg-red-900/50 text-red-300 border border-red-800/60 text-xs font-medium transition-colors"
                >
                  Delete Organization
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* MODAL: INVITE MEMBER */}
      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="card max-w-md w-full p-6 space-y-4 animate-in fade-in zoom-in-95 duration-150">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <UserPlus className="w-5 h-5 text-forge-accent" />
              Invite Member to {org.name}
            </h3>

            {inviteError && (
              <div className="p-3 bg-red-950/40 border border-red-800/50 rounded-lg text-red-200 text-xs">
                {inviteError}
              </div>
            )}

            <form onSubmit={handleInvite} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Username <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. developer1"
                  value={inviteUsername}
                  onChange={(e) => setInviteUsername(e.target.value)}
                  className="input-field"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Organization Role
                </label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value as OrgRole)}
                  className="input-field capitalize"
                >
                  <option value="member">Member (Regular access)</option>
                  <option value="admin">Admin (Manage members & teams)</option>
                  {isOwner && <option value="owner">Owner (Full administrative control)</option>}
                  <option value="billing_manager">Billing Manager</option>
                </select>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-forge-border">
                <button
                  type="button"
                  onClick={() => setShowInviteModal(false)}
                  className="btn-secondary text-xs"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={inviteLoading}
                  className="btn-primary text-xs"
                >
                  {inviteLoading ? 'Adding...' : 'Add Member'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* MODAL: CREATE TEAM */}
      {showTeamModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="card max-w-md w-full p-6 space-y-4 animate-in fade-in zoom-in-95 duration-150">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <Users className="w-5 h-5 text-forge-accent" />
              Create Team in {org.name}
            </h3>

            {teamError && (
              <div className="p-3 bg-red-950/40 border border-red-800/50 rounded-lg text-red-200 text-xs">
                {teamError}
              </div>
            )}

            <form onSubmit={handleCreateTeam} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Team Name <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Core Engineering"
                  value={teamName}
                  onChange={(e) => {
                    setTeamName(e.target.value);
                    if (!teamSlug) {
                      setTeamSlug(
                        e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, '-').slice(0, 39)
                      );
                    }
                  }}
                  className="input-field"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Team Slug <span className="text-red-400">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="core-engineering"
                  value={teamSlug}
                  onChange={(e) => setTeamSlug(e.target.value.toLowerCase().replace(/[^a-z0-9_-]/g, ''))}
                  className="input-field font-mono text-xs"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Description
                </label>
                <textarea
                  rows={2}
                  placeholder="What is this team's focus?"
                  value={teamDesc}
                  onChange={(e) => setTeamDesc(e.target.value)}
                  className="input-field resize-none"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Visibility
                </label>
                <select
                  value={teamPrivacy}
                  onChange={(e) => setTeamPrivacy(e.target.value as 'visible' | 'secret')}
                  className="input-field"
                >
                  <option value="visible">Visible (All org members can see this team)</option>
                  <option value="secret">Secret (Only members of this team can see it)</option>
                </select>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-forge-border">
                <button
                  type="button"
                  onClick={() => setShowTeamModal(false)}
                  className="btn-secondary text-xs"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={teamLoading}
                  className="btn-primary text-xs"
                >
                  {teamLoading ? 'Creating...' : 'Create Team'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
