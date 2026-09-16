export interface DatabaseHealth {
  status: 'healthy' | 'down' | 'standalone' | 'unconfigured';
  latency?: number;
  open_conns?: number;
  in_use_conns?: number;
  idle_conns?: number;
  driver?: string;
  error?: string;
}

export interface RedisHealth {
  status: 'healthy' | 'down' | 'unconfigured';
  latency?: number;
  mode?: string;
  error?: string;
}

export interface SystemMetrics {
  go_version: string;
  num_goroutine: number;
  num_cpu: number;
  alloc_mb: number;
  total_alloc_mb: number;
  sys_mb: number;
}

export interface TelemetryResponse {
  status: 'healthy' | 'degraded' | 'down';
  version: string;
  uptime_seconds: number;
  timestamp: string;
  database: DatabaseHealth;
  redis: RedisHealth;
  system: SystemMetrics;
}

export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  avatar_url: string;
  bio?: string;
  location?: string;
  website?: string;
  is_admin: boolean;
  is_suspended: boolean;
  created_at: string;
  updated_at?: string;
}

export interface PublicUser {
  id: string;
  username: string;
  display_name: string;
  avatar_url: string;
  bio?: string;
  location?: string;
  website?: string;
  created_at: string;
}

export interface APIToken {
  id: string;
  user_id: string;
  name: string;
  token_prefix: string;
  scopes: string[];
  last_used_at?: string;
  expires_at?: string;
  created_at: string;
}

export interface Repository {
  id: string;
  name: string;
  slug: string;
  description: string;
  visibility: 'public' | 'private' | 'internal';
  default_branch: string;
  is_archived?: boolean;
  is_fork?: boolean;
  forked_from_id?: string;
  owner_type?: 'user' | 'org';
  owner_user_id?: string;
  owner_org_id?: string;
  owner_name: string;
  clone_url?: string;
  http_clone_url?: string;
  stars_count: number;
  forks_count: number;
  created_at?: string;
  updated_at: string;
  current_user_permission?: RepoPermission;
}

export type OrgRole = 'owner' | 'admin' | 'member' | 'billing_manager';
export type TeamRole = 'maintainer' | 'member';
export type RepoPermission = 'read' | 'triage' | 'write' | 'maintain' | 'admin';

export interface Organization {
  id: string;
  name: string;
  slug: string;
  description: string;
  avatar_url: string;
  website?: string;
  location?: string;
  created_at: string;
  updated_at: string;
  member_count?: number;
  team_count?: number;
  repo_count?: number;
  role?: OrgRole;
}

export interface OrgMember {
  id: string;
  organization_id: string;
  user_id: string;
  username: string;
  display_name: string;
  avatar_url: string;
  email: string;
  role: OrgRole;
  created_at: string;
  updated_at: string;
}

export interface Team {
  id: string;
  organization_id: string;
  name: string;
  slug: string;
  description: string;
  privacy: 'visible' | 'secret';
  member_count: number;
  created_at: string;
  updated_at: string;
  current_user_role?: TeamRole;
}

export interface TeamMember {
  id: string;
  team_id: string;
  user_id: string;
  username: string;
  display_name: string;
  avatar_url: string;
  role: TeamRole;
  created_at: string;
}

