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

export interface Label {
  id: string;
  repository_id: string;
  name: string;
  color: string;
  description: string;
  created_at: string;
}

export interface Milestone {
  id: string;
  repository_id: string;
  title: string;
  description: string;
  state: 'open' | 'closed';
  due_date?: string;
  open_issues_count?: number;
  closed_issues_count?: number;
  created_at: string;
  updated_at: string;
  closed_at?: string;
}

export interface Issue {
  id: string;
  repository_id: string;
  number: number;
  title: string;
  body: string;
  state: 'open' | 'closed';
  author_id?: string;
  author?: User;
  milestone_id?: string;
  milestone?: Milestone;
  labels: Label[];
  assignees: User[];
  comments_count: number;
  closed_at?: string;
  closed_by_id?: string;
  closed_by?: User;
  created_at: string;
  updated_at: string;
}

export interface IssueComment {
  id: string;
  issue_id: string;
  author_id?: string;
  author?: User;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface IssueDetail extends Issue {
  comments: IssueComment[];
}

export interface GitCommit {
  sha: string;
  short_sha: string;
  author_name: string;
  author_email: string;
  committer_name: string;
  committer_email: string;
  message: string;
  date: string;
}

export type ReviewState = 'APPROVED' | 'CHANGES_REQUESTED' | 'COMMENTED' | 'PENDING';

export interface PullRequest {
  id: string;
  repository_id: string;
  number: number;
  title: string;
  body: string;
  state: 'open' | 'closed' | 'merged';
  is_draft: boolean;
  source_branch: string;
  target_branch: string;
  author_id?: string;
  author?: User | PublicUser;
  merge_commit_sha?: string;
  merged_by_id?: string;
  merged_by?: User | PublicUser;
  merged_at?: string;
  closed_at?: string;
  comments_count: number;
  reviews_count: number;
  created_at: string;
  updated_at: string;
}

export interface PullRequestReview {
  id: string;
  pull_request_id: string;
  reviewer_id: string;
  reviewer?: User | PublicUser;
  state: ReviewState;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface PullRequestComment {
  id: string;
  pull_request_id: string;
  review_id?: string;
  author_id?: string;
  author?: User | PublicUser;
  file_path?: string;
  line_number?: number;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface DiffLine {
  type: 'context' | 'addition' | 'deletion';
  content: string;
  old_line_no: number;
  new_line_no: number;
}

export interface DiffHunk {
  old_start: number;
  old_lines: number;
  new_start: number;
  new_lines: number;
  header: string;
  lines: DiffLine[];
}

export interface DiffFile {
  old_path: string;
  new_path: string;
  status: 'added' | 'modified' | 'deleted' | 'renamed';
  additions: number;
  deletions: number;
  hunks: DiffHunk[];
}

export interface DiffResult {
  files: DiffFile[];
  total_files: number;
  total_additions: number;
  total_deletions: number;
}

export interface PullRequestDetail extends PullRequest {
  reviews: PullRequestReview[];
  comments: PullRequestComment[];
  commits: GitCommit[];
  diff?: DiffResult;
  can_merge: boolean;
  has_conflicts: boolean;
}

export type PipelineStatus = 'queued' | 'running' | 'success' | 'failure' | 'cancelled';
export type JobStatus = 'queued' | 'running' | 'success' | 'failure' | 'skipped' | 'cancelled';
export type StepStatus = 'pending' | 'running' | 'success' | 'failure' | 'skipped';

export interface PipelineRun {
  id: string;
  repository_id: string;
  workflow_name: string;
  workflow_file: string;
  trigger_event: string;
  trigger_user_id?: string;
  trigger_user?: PublicUser;
  commit_sha: string;
  commit_message: string;
  branch: string;
  status: PipelineStatus;
  started_at?: string;
  finished_at?: string;
  duration_ms: number;
  created_at: string;
  updated_at: string;
}

export interface PipelineJob {
  id: string;
  pipeline_run_id: string;
  job_key: string;
  name: string;
  runs_on: string;
  status: JobStatus;
  started_at?: string;
  finished_at?: string;
  duration_ms: number;
  exit_code?: number;
  created_at: string;
  updated_at: string;
  steps?: JobStep[];
}

export interface JobStep {
  id: string;
  pipeline_job_id: string;
  step_number: number;
  name: string;
  command: string;
  status: StepStatus;
  started_at?: string;
  finished_at?: string;
  duration_ms: number;
  exit_code?: number;
  created_at: string;
}

export interface JobLog {
  id: string;
  pipeline_job_id: string;
  step_id?: string;
  line_number: number;
  content: string;
  stream: 'stdout' | 'stderr' | 'system';
  created_at: string;
}

export interface PipelineRunDetail extends PipelineRun {
  jobs: PipelineJob[];
}


