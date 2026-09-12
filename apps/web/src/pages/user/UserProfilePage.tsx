import React from 'react';
import { useParams, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import {
  MapPin,
  Link as LinkIcon,
  Calendar,
  FolderGit2,
  Star,
  Activity,
  Edit3,
  HelpCircle
} from 'lucide-react';
import { PublicUser } from '../../types';

export const UserProfilePage: React.FC = () => {
  const { username } = useParams<{ username: string }>();
  const { user: currentUser } = useAuth();

  const isOwnProfile = currentUser && currentUser.username.toLowerCase() === username?.toLowerCase();

  const {
    data: profile,
    isLoading,
    isError,
  } = useQuery<PublicUser>({
    queryKey: ['user-profile', username],
    queryFn: async () => {
      const res = await fetch(`/api/v1/users/${username}`);
      if (!res.ok) {
        throw new Error('User not found');
      }
      const data = await res.json();
      return data.user;
    },
    enabled: !!username,
  });

  if (isLoading) {
    return (
      <div className="py-20 text-center text-xs text-forge-muted">
        Loading developer profile...
      </div>
    );
  }

  if (isError || !profile) {
    return (
      <div className="py-20 text-center space-y-4">
        <div className="w-14 h-14 rounded-2xl bg-forge-card border border-forge-border flex items-center justify-center mx-auto text-forge-muted">
          <HelpCircle className="w-7 h-7" />
        </div>
        <h2 className="text-xl font-bold text-white">User @{username} not found</h2>
        <p className="text-xs text-forge-muted">The developer profile you are looking for does not exist.</p>
        <Link to="/dashboard" className="text-xs text-blue-400 hover:underline">
          Return to Dashboard
        </Link>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
      {/* Sidebar: Profile Info */}
      <div className="space-y-5">
        {/* Avatar */}
        <div className="w-48 h-48 rounded-full bg-gradient-to-tr from-indigo-600 via-blue-600 to-cyan-500 border-4 border-forge-border/80 flex items-center justify-center text-5xl font-bold text-white shadow-2xl overflow-hidden">
          {profile.avatar_url ? (
            <img src={profile.avatar_url} alt={profile.username} className="w-full h-full object-cover" />
          ) : (
            profile.username.slice(0, 2).toUpperCase()
          )}
        </div>

        {/* Names */}
        <div className="space-y-0.5">
          <h1 className="text-xl font-bold text-white">
            {profile.display_name || profile.username}
          </h1>
          <p className="text-sm font-mono text-forge-muted">@{profile.username}</p>
        </div>

        {/* Bio */}
        {profile.bio && (
          <p className="text-xs text-forge-text leading-relaxed">{profile.bio}</p>
        )}

        {/* Edit Button */}
        {isOwnProfile && (
          <Link
            to="/settings/profile"
            className="w-full py-1.5 px-3 rounded-lg bg-forge-card hover:bg-forge-subtle border border-forge-border text-xs font-semibold text-forge-text flex items-center justify-center space-x-1.5 transition-colors"
          >
            <Edit3 className="w-3.5 h-3.5 text-forge-muted" />
            <span>Edit Profile</span>
          </Link>
        )}

        {/* Metadata */}
        <div className="space-y-2 text-xs text-forge-muted pt-2 border-t border-forge-border/40">
          {profile.location && (
            <div className="flex items-center space-x-2">
              <MapPin className="w-3.5 h-3.5" />
              <span>{profile.location}</span>
            </div>
          )}
          {profile.website && (
            <div className="flex items-center space-x-2">
              <LinkIcon className="w-3.5 h-3.5" />
              <a href={profile.website} target="_blank" rel="noreferrer" className="text-blue-400 hover:underline truncate">
                {profile.website}
              </a>
            </div>
          )}
          <div className="flex items-center space-x-2">
            <Calendar className="w-3.5 h-3.5" />
            <span>Joined {new Date(profile.created_at).toLocaleDateString()}</span>
          </div>
        </div>
      </div>

      {/* Main Content Area: Tabs & Content */}
      <div className="md:col-span-3 space-y-6">
        {/* Navigation Tabs */}
        <div className="flex items-center space-x-6 border-b border-forge-border pb-2 text-xs font-medium">
          <button className="flex items-center space-x-2 pb-2 -mb-2 border-b-2 border-blue-500 text-white font-semibold">
            <FolderGit2 className="w-4 h-4 text-blue-400" />
            <span>Repositories</span>
            <span className="px-1.5 py-0.2 rounded-full bg-forge-card text-[11px] text-forge-muted font-mono">
              0
            </span>
          </button>
          <button className="flex items-center space-x-2 text-forge-muted hover:text-white transition-colors">
            <Star className="w-4 h-4" />
            <span>Stars</span>
            <span className="px-1.5 py-0.2 rounded-full bg-forge-card text-[11px] text-forge-muted font-mono">
              0
            </span>
          </button>
          <button className="flex items-center space-x-2 text-forge-muted hover:text-white transition-colors">
            <Activity className="w-4 h-4" />
            <span>Activity</span>
          </button>
        </div>

        {/* Repositories Tab Content */}
        <div className="glass-panel p-10 rounded-xl text-center space-y-3">
          <div className="w-12 h-12 rounded-xl bg-forge-card border border-forge-border flex items-center justify-center mx-auto text-forge-muted">
            <FolderGit2 className="w-6 h-6" />
          </div>
          <h3 className="text-sm font-semibold text-white">No public repositories</h3>
          <p className="text-xs text-forge-muted max-w-sm mx-auto">
            {profile.username} does not have any public repositories yet.
          </p>
        </div>
      </div>
    </div>
  );
};
