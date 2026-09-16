-- 000006_webhooks_and_deliveries.up.sql
-- ForgeHub Webhooks, Event Subscriptions, and Delivery Auditing

CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    content_type VARCHAR(50) NOT NULL DEFAULT 'application/json',
    secret VARCHAR(255),
    events TEXT[] NOT NULL DEFAULT '{"push"}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ssl_verification BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event VARCHAR(50) NOT NULL,
    action VARCHAR(50),
    request_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_status_code INT,
    response_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_body TEXT,
    duration_ms INT NOT NULL DEFAULT 0,
    is_success BOOLEAN NOT NULL DEFAULT FALSE,
    error_message TEXT,
    delivered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_repository ON webhooks(repository_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks(repository_id, is_active);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_hook_delivered ON webhook_deliveries(webhook_id, delivered_at DESC);
