-- ForgeHub Phase 3 Schema Migration: Org enhancements and Team Repository permissions

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS website VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS location VARCHAR(128) NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS team_repositories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    permission VARCHAR(32) NOT NULL CHECK (permission IN ('read', 'triage', 'write', 'maintain', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(team_id, repository_id)
);

CREATE INDEX IF NOT EXISTS idx_team_repos_team ON team_repositories(team_id);
CREATE INDEX IF NOT EXISTS idx_team_repos_repo ON team_repositories(repository_id);
