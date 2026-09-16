-- ============================================================================
-- PULL REQUESTS & CODE REVIEW
-- ============================================================================
CREATE TABLE IF NOT EXISTS pull_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number INT NOT NULL,
    title VARCHAR(256) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    state VARCHAR(16) NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'closed', 'merged')),
    source_branch VARCHAR(128) NOT NULL,
    target_branch VARCHAR(128) NOT NULL,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    is_draft BOOLEAN NOT NULL DEFAULT FALSE,
    merge_commit_sha VARCHAR(64),
    merged_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    merged_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    closed_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_repo_pr_number UNIQUE (repository_id, number)
);

CREATE INDEX IF NOT EXISTS idx_pull_requests_repo ON pull_requests(repository_id);
CREATE INDEX IF NOT EXISTS idx_pull_requests_number ON pull_requests(repository_id, number);
CREATE INDEX IF NOT EXISTS idx_pull_requests_state ON pull_requests(state);
CREATE INDEX IF NOT EXISTS idx_pull_requests_author ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_pull_requests_branches ON pull_requests(repository_id, source_branch, target_branch);
CREATE INDEX IF NOT EXISTS idx_pull_requests_created ON pull_requests(created_at DESC);

-- PULL REQUEST REVIEWS
CREATE TABLE IF NOT EXISTS pull_request_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state VARCHAR(32) NOT NULL CHECK (state IN ('commented', 'approved', 'changes_requested')),
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pr_reviews_pr ON pull_request_reviews(pull_request_id);
CREATE INDEX IF NOT EXISTS idx_pr_reviews_reviewer ON pull_request_reviews(reviewer_id);

-- PULL REQUEST COMMENTS
CREATE TABLE IF NOT EXISTS pull_request_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    review_id UUID REFERENCES pull_request_reviews(id) ON DELETE CASCADE,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    file_path VARCHAR(512),
    line_number INT,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pr_comments_pr ON pull_request_comments(pull_request_id);
CREATE INDEX IF NOT EXISTS idx_pr_comments_author ON pull_request_comments(author_id);
CREATE INDEX IF NOT EXISTS idx_pr_comments_created ON pull_request_comments(created_at ASC);
