import React, { useState, useEffect } from 'react';
import { Navigate, Link } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import { User, Lock, Key, Check, AlertCircle, Save } from 'lucide-react';

export const ProfileSettingsPage: React.FC = () => {
  const { user, isAuthenticated, isLoading, refetchUser } = useAuth();

  const [displayName, setDisplayName] = useState('');
  const [bio, setBio] = useState('');
  const [location, setLocation] = useState('');
  const [website, setWebsite] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');

  const [profileMsg, setProfileMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [isSavingProfile, setIsSavingProfile] = useState(false);

  // Password change state
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [passwordMsg, setPasswordMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [isChangingPassword, setIsChangingPassword] = useState(false);

  useEffect(() => {
    if (user) {
      setDisplayName(user.display_name || '');
      setBio(user.bio || '');
      setLocation(user.location || '');
      setWebsite(user.website || '');
      setAvatarUrl(user.avatar_url || '');
    }
  }, [user]);

  if (isLoading) {
    return <div className="py-12 text-center text-xs text-forge-muted">Loading settings...</div>;
  }

  if (!isAuthenticated || !user) {
    return <Navigate to="/login" replace />;
  }

  const handleSaveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setIsSavingProfile(true);
      setProfileMsg(null);
      const res = await fetch('/api/v1/users/me', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          display_name: displayName,
          bio: bio,
          location: location,
          website: website,
          avatar_url: avatarUrl,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to update profile');
      }
      setProfileMsg({ type: 'success', text: 'Profile updated successfully!' });
      await refetchUser();
    } catch (err: any) {
      setProfileMsg({ type: 'error', text: err.message || 'Error updating profile' });
    } finally {
      setIsSavingProfile(false);
    }
  };

  const handleChangePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setPasswordMsg({ type: 'error', text: 'New passwords do not match' });
      return;
    }
    try {
      setIsChangingPassword(true);
      setPasswordMsg(null);
      const res = await fetch('/api/v1/users/me/password', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || 'Failed to change password');
      }
      setPasswordMsg({ type: 'success', text: 'Password changed successfully!' });
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
    } catch (err: any) {
      setPasswordMsg({ type: 'error', text: err.message || 'Error changing password' });
    } finally {
      setIsChangingPassword(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Settings Header */}
      <div className="pb-4 border-b border-forge-border">
        <h1 className="text-2xl font-bold tracking-tight text-white">Developer Settings</h1>
        <p className="text-xs text-forge-muted mt-1">
          Manage your personal profile information, security credentials, and access tokens.
        </p>
      </div>

      {/* Tabs */}
      <div className="flex items-center space-x-6 border-b border-forge-border pb-2 text-xs font-medium">
        <Link to="/settings/profile" className="flex items-center space-x-2 pb-2 -mb-2 border-b-2 border-blue-500 text-white font-semibold">
          <User className="w-4 h-4 text-blue-400" />
          <span>Public Profile</span>
        </Link>
        <Link to="/settings/tokens" className="flex items-center space-x-2 text-forge-muted hover:text-white transition-colors">
          <Key className="w-4 h-4" />
          <span>Personal Access Tokens</span>
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Profile Card */}
        <div className="glass-panel p-6 rounded-xl space-y-5">
          <h2 className="text-sm font-semibold text-white flex items-center gap-2">
            <User className="w-4 h-4 text-blue-400" />
            <span>Public Profile Details</span>
          </h2>

          {profileMsg && (
            <div className={`p-3 rounded-lg text-xs flex items-center space-x-2 ${
              profileMsg.type === 'success' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
            }`}>
              {profileMsg.type === 'success' ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
              <span>{profileMsg.text}</span>
            </div>
          )}

          <form onSubmit={handleSaveProfile} className="space-y-4">
            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">Display Name</label>
              <input
                type="text"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Your full or display name"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">Bio</label>
              <textarea
                rows={3}
                value={bio}
                onChange={(e) => setBio(e.target.value)}
                placeholder="Tell us a little bit about yourself"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">Location</label>
              <input
                type="text"
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                placeholder="e.g. San Francisco, CA"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">Website or Portfolio</label>
              <input
                type="url"
                value={website}
                onChange={(e) => setWebsite(e.target.value)}
                placeholder="https://yourportfolio.dev"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <button
              type="submit"
              disabled={isSavingProfile}
              className="py-1.5 px-4 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold flex items-center space-x-1.5 shadow-md transition-all"
            >
              <Save className="w-3.5 h-3.5" />
              <span>{isSavingProfile ? 'Saving...' : 'Update Profile'}</span>
            </button>
          </form>
        </div>

        {/* Change Password Card */}
        <div className="glass-panel p-6 rounded-xl space-y-5">
          <h2 className="text-sm font-semibold text-white flex items-center gap-2">
            <Lock className="w-4 h-4 text-purple-400" />
            <span>Change Password</span>
          </h2>

          {passwordMsg && (
            <div className={`p-3 rounded-lg text-xs flex items-center space-x-2 ${
              passwordMsg.type === 'success' ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20' : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
            }`}>
              {passwordMsg.type === 'success' ? <Check className="w-4 h-4" /> : <AlertCircle className="w-4 h-4" />}
              <span>{passwordMsg.text}</span>
            </div>
          )}

          <form onSubmit={handleChangePassword} className="space-y-4">
            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">Current Password</label>
              <input
                type="password"
                required
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                placeholder="Enter current password"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">New Password</label>
              <input
                type="password"
                required
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="At least 8 chars, uppercase, lowercase, number"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium text-forge-muted">Confirm New Password</label>
              <input
                type="password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder="Re-enter new password"
                className="w-full px-3 py-1.5 bg-forge-card border border-forge-border rounded-lg text-xs text-forge-text focus:outline-none focus:border-blue-500"
              />
            </div>

            <button
              type="submit"
              disabled={isChangingPassword}
              className="py-1.5 px-4 rounded-lg bg-forge-card hover:bg-forge-subtle border border-forge-border text-white text-xs font-semibold flex items-center space-x-1.5 transition-all"
            >
              <Lock className="w-3.5 h-3.5 text-purple-400" />
              <span>{isChangingPassword ? 'Updating...' : 'Update Password'}</span>
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};
