-- 000005_pipelines_and_runners.up.sql
-- ForgeHub CI/CD Pipelines, Jobs, Steps, and Execution Logs

CREATE TABLE IF NOT EXISTS pipeline_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    workflow_name VARCHAR(100) NOT NULL,
    workflow_file VARCHAR(255) NOT NULL,
    trigger_event VARCHAR(50) NOT NULL,
    trigger_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    commit_sha VARCHAR(40) NOT NULL,
    commit_message TEXT,
    branch VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS pipeline_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_run_id UUID NOT NULL REFERENCES pipeline_runs(id) ON DELETE CASCADE,
    job_key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    runs_on VARCHAR(100) NOT NULL DEFAULT 'ubuntu-latest',
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    exit_code INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS job_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_job_id UUID NOT NULL REFERENCES pipeline_jobs(id) ON DELETE CASCADE,
    step_number INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    command TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    exit_code INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS job_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_job_id UUID NOT NULL REFERENCES pipeline_jobs(id) ON DELETE CASCADE,
    step_id UUID REFERENCES job_steps(id) ON DELETE CASCADE,
    line_number INT NOT NULL,
    content TEXT NOT NULL,
    stream VARCHAR(20) NOT NULL DEFAULT 'stdout',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for efficient lookups & queries
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_repo_created ON pipeline_runs(repository_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_repo_status ON pipeline_runs(repository_id, status);
CREATE INDEX IF NOT EXISTS idx_pipeline_jobs_run ON pipeline_jobs(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_job_steps_job ON job_steps(pipeline_job_id, step_number);
CREATE INDEX IF NOT EXISTS idx_job_logs_job_line ON job_logs(pipeline_job_id, line_number);
