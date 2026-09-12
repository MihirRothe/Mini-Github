import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { Team, TeamMember, TeamRole, OrgMember } from '../../types';
import {
  Users,
  UserPlus,
  Trash2,
  ArrowLeft,
  Loader2
} from 'lucide-react';

export const TeamDetailPage: React.FC = () => {
  const { org: orgSlug, team: teamSlug } = useParams<{ org: string; team: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [team, setTeam] = useState<Team | null>(null);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [orgMembers, setOrgMembers] = useState<OrgMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Add Member Modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedUsername, setSelectedUsername] = useState('');
  const [selectedRole, setSelectedRole] = useState<TeamRole>('member');
  const [addLoading, setAddLoading] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);

  const fetchTeamData = async () => {
    if (!orgSlug || !teamSlug) return;
    setLoading(true);
    setError(null);

    try {
      // 1. Fetch team
      const res = await fetch(`/api/v1/orgs/${orgSlug}/teams/${teamSlug}`, { credentials: 'include' });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Team not found.');
      }
      setTeam(data.team);

      // 2. Fetch team members
      const resMembers = await fetch(`/api/v1/orgs/${orgSlug}/teams/${teamSlug}/members`, {
        credentials: 'include',
      });
      if (resMembers.ok) {
        const memData = await resMembers.json();
        setMembers(memData.members || []);
      }

      // 3. Fetch org members for autocomplete
      const resOrgMembers = await fetch(`/api/v1/orgs/${orgSlug}/members`, {
        credentials: 'include',
      });
      if (resOrgMembers.ok) {
        const orgMemData = await resOrgMembers.json();
        setOrgMembers(orgMemData.members || []);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load team.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTeamData();
  }, [orgSlug, teamSlug]);

  const canManage =
    team?.current_user_role === 'maintainer' ||
    user?.is_admin;

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedUsername.trim()) return;
    setAddLoading(true);
    setAddError(null);

    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/teams/${teamSlug}/members`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          username: selectedUsername.trim(),
          role: selectedRole,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to add member to team.');
      }
      setMembers((prev) => [...prev, data.member]);
      setShowAddModal(false);
      setSelectedUsername('');
    } catch (err: any) {
      setAddError(err.message);
    } finally {
      setAddLoading(false);
    }
  };

  const handleRoleChange = async (username: string, newRole: TeamRole) => {
    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/teams/${teamSlug}/members/${username}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ role: newRole }),
      });
      if (!res.ok) {
        const data = await res.json();
        alert(data.error?.message || 'Failed to update role.');
        return;
      }
      setMembers((prev) =>
        prev.map((m) => (m.username === username ? { ...m, role: newRole } : m))
      );
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleRemoveMember = async (username: string) => {
    if (!confirm(`Are you sure you want to remove @${username} from team ${team?.name}?`)) return;
    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/teams/${teamSlug}/members/${username}`, {
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

  const handleDeleteTeam = async () => {
    if (!confirm(`Delete team "${team?.name}" permanently?`)) return;
    try {
      const res = await fetch(`/api/v1/orgs/${orgSlug}/teams/${teamSlug}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (!res.ok) {
        const data = await res.json();
        alert(data.error?.message || 'Failed to delete team.');
        return;
      }
      navigate(`/orgs/${orgSlug}`);
    } catch (err: any) {
      alert(err.message);
    }
  };

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto py-16 px-4 text-center">
        <Loader2 className="w-8 h-8 text-forge-accent animate-spin mx-auto mb-3" />
        <p className="text-forge-muted text-sm">Loading team details...</p>
      </div>
    );
  }

  if (error || !team) {
    return (
      <div className="max-w-2xl mx-auto py-16 px-4 text-center">
        <Users className="w-16 h-16 mx-auto text-forge-muted mb-4" />
        <h2 className="text-xl font-bold text-white mb-2">Team Not Found</h2>
        <p className="text-forge-muted mb-6">{error || 'The requested team does not exist.'}</p>
        <Link to={`/orgs/${orgSlug}`} className="btn-secondary">Back to Organization</Link>
      </div>
    );
  }

  // Filter org members not already in the team
  const existingUsernames = new Set(members.map((m) => m.username));
  const availableOrgMembers = orgMembers.filter((m) => !existingUsernames.has(m.username));

  return (
    <div className="max-w-5xl mx-auto py-8 px-4 sm:px-6">
      {/* Breadcrumb back */}
      <div className="mb-6">
        <Link
          to={`/orgs/${orgSlug}`}
          className="inline-flex items-center gap-1.5 text-xs text-forge-muted hover:text-white transition-colors"
        >
          <ArrowLeft className="w-3.5 h-3.5" />
          Back to @{orgSlug}
        </Link>
      </div>

      {/* Team Header Banner */}
      <div className="card p-6 sm:p-8 mb-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <div className="w-14 h-14 rounded-2xl bg-forge-accent/10 border border-forge-accent/20 flex items-center justify-center text-forge-accent">
              <Users className="w-8 h-8" />
            </div>
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold text-white">{team.name}</h1>
                <span className="px-2.5 py-0.5 rounded text-[10px] font-semibold uppercase bg-forge-border text-forge-muted">
                  {team.privacy}
                </span>
                {team.current_user_role && (
                  <span className="px-2 py-0.5 rounded-full text-xs font-semibold bg-forge-accent/20 text-forge-accent capitalize">
                    {team.current_user_role}
                  </span>
                )}
              </div>
              <div className="text-xs font-mono text-forge-muted mt-0.5">
                @{orgSlug}/{team.slug}
              </div>
              {team.description && (
                <p className="text-xs text-forge-muted mt-2 max-w-xl">{team.description}</p>
              )}
            </div>
          </div>

          <div className="flex items-center gap-3 self-start sm:self-auto">
            {canManage && (
              <>
                <button
                  onClick={() => setShowAddModal(true)}
                  className="btn-primary text-xs flex items-center gap-1.5"
                >
                  <UserPlus className="w-4 h-4" />
                  Add Member
                </button>
                <button
                  onClick={handleDeleteTeam}
                  className="p-2 text-forge-muted hover:text-red-400 border border-forge-border hover:border-red-800/50 rounded-lg transition-colors"
                  title="Delete Team"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Team Members Roster */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-white flex items-center gap-2">
            <Users className="w-5 h-5 text-forge-accent" />
            Team Members ({members.length})
          </h2>
        </div>

        <div className="card divide-y divide-forge-border overflow-hidden">
          {members.map((member) => (
            <div key={member.id} className="p-4 sm:px-6 flex items-center justify-between gap-4">
              <div className="flex items-center gap-3">
                <img
                  src={member.avatar_url || `https://api.dicebear.com/7.x/identicon/svg?seed=${member.username}`}
                  alt={member.username}
                  className="w-9 h-9 rounded-full bg-forge-bg border border-forge-border object-cover"
                />
                <div>
                  <div className="flex items-center gap-2">
                    <Link
                      to={`/${member.username}`}
                      className="text-sm font-medium text-white hover:text-forge-accent"
                    >
                      {member.display_name || member.username}
                    </Link>
                    <span className="text-xs font-mono text-forge-muted">@{member.username}</span>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-4">
                {canManage && member.username !== user?.username ? (
                  <select
                    value={member.role}
                    onChange={(e) => handleRoleChange(member.username, e.target.value as TeamRole)}
                    className="bg-forge-bg border border-forge-border rounded-lg text-xs text-white px-2.5 py-1 focus:outline-none focus:border-forge-accent capitalize"
                  >
                    <option value="member">Member</option>
                    <option value="maintainer">Maintainer</option>
                  </select>
                ) : (
                  <span className="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-forge-border text-forge-text capitalize">
                    {member.role}
                  </span>
                )}

                {(canManage || member.username === user?.username) && (
                  <button
                    onClick={() => handleRemoveMember(member.username)}
                    title={member.username === user?.username ? "Leave Team" : "Remove Member"}
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

      {/* MODAL: ADD TEAM MEMBER */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className="card max-w-md w-full p-6 space-y-4 animate-in fade-in zoom-in-95 duration-150">
            <h3 className="text-lg font-bold text-white flex items-center gap-2">
              <UserPlus className="w-5 h-5 text-forge-accent" />
              Add Member to {team.name}
            </h3>

            {addError && (
              <div className="p-3 bg-red-950/40 border border-red-800/50 rounded-lg text-red-200 text-xs">
                {addError}
              </div>
            )}

            <form onSubmit={handleAddMember} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Select Organization Member
                </label>
                {availableOrgMembers.length > 0 ? (
                  <select
                    value={selectedUsername}
                    onChange={(e) => setSelectedUsername(e.target.value)}
                    required
                    className="input-field"
                  >
                    <option value="">-- Choose member --</option>
                    {availableOrgMembers.map((m) => (
                      <option key={m.id} value={m.username}>
                        {m.display_name ? `${m.display_name} (@${m.username})` : `@${m.username}`}
                      </option>
                    ))}
                  </select>
                ) : (
                  <div className="p-3 bg-forge-bg rounded-lg border border-forge-border text-xs text-forge-muted">
                    All members of {orgSlug} are already in this team!
                  </div>
                )}
              </div>

              <div>
                <label className="block text-xs font-semibold text-forge-muted uppercase mb-1">
                  Team Role
                </label>
                <select
                  value={selectedRole}
                  onChange={(e) => setSelectedRole(e.target.value as TeamRole)}
                  className="input-field capitalize"
                >
                  <option value="member">Member</option>
                  <option value="maintainer">Maintainer (Can manage team members & settings)</option>
                </select>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-forge-border">
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  className="btn-secondary text-xs"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={addLoading || !selectedUsername}
                  className="btn-primary text-xs"
                >
                  {addLoading ? 'Adding...' : 'Add to Team'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
