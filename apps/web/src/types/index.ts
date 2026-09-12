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
  is_admin: boolean;
  created_at: string;
}

export interface Repository {
  id: string;
  name: string;
  slug: string;
  description: string;
  visibility: 'public' | 'private' | 'internal';
  default_branch: string;
  owner_name: string;
  stars_count: number;
  forks_count: number;
  updated_at: string;
}
